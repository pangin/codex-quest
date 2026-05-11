package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type codexEventMsg struct {
	Type string `json:"type"`

	Message string               `json:"message,omitempty"`
	Info    *codexTokenCountInfo `json:"info,omitempty"`
}

type codexTokenCountInfo struct {
	TotalTokenUsage    *TokenUsage `json:"total_token_usage,omitempty"`
	LastTokenUsage     *TokenUsage `json:"last_token_usage,omitempty"`
	ModelContextWindow int         `json:"model_context_window,omitempty"`
}

type codexResponseItem struct {
	Type      string          `json:"type"`
	Role      string          `json:"role,omitempty"`
	Name      string          `json:"name,omitempty"`
	CallID    string          `json:"call_id,omitempty"`
	Status    string          `json:"status,omitempty"`
	Action    json.RawMessage `json:"action,omitempty"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
	Output    string          `json:"output,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
	Summary   json.RawMessage `json:"summary,omitempty"`
}

type codexToolArgs struct {
	Cmd         string          `json:"cmd,omitempty"`
	Command     string          `json:"command,omitempty"`
	ToolUses    []codexToolArgs `json:"tool_uses,omitempty"`
	Plan        []codexPlanItem `json:"plan,omitempty"`
	AgentType   string          `json:"agent_type,omitempty"`
	Description string          `json:"description,omitempty"`
}

type codexPlanItem struct {
	Step   string `json:"step"`
	Status string `json:"status"`
}

func (w *Watcher) parseCodexLine(line string) ([]Event, bool) {
	var record codexRecord
	if err := json.Unmarshal([]byte(line), &record); err != nil {
		return nil, false
	}

	switch record.Type {
	case "session_meta":
		var meta codexSessionMeta
		if err := json.Unmarshal(record.Payload, &meta); err != nil {
			return nil, true
		}
		details := "Codex session started"
		if meta.Cwd != "" {
			details = "Session: " + meta.Cwd
		}
		return []Event{{Type: EventSystemInit, Details: details}}, true

	case "event_msg":
		return w.parseCodexEventMsg(record.Payload), true

	case "response_item":
		return w.parseCodexResponseItem(record.Payload), true

	case "turn_context":
		return nil, true

	default:
		return nil, false
	}
}

func (w *Watcher) parseCodexEventMsg(raw json.RawMessage) []Event {
	var msg codexEventMsg
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil
	}

	switch msg.Type {
	case "task_started":
		return []Event{{Type: EventSystemInit, Details: "Codex task started"}}

	case "user_message":
		text := strings.TrimSpace(msg.Message)
		if text == "" {
			return nil
		}
		if thinkLevel := detectThinkLevel(text); thinkLevel != ThinkNone {
			return []Event{{
				Type:       EventThinkHard,
				Details:    truncate(text, 50),
				ThinkLevel: thinkLevel,
			}}
		}
		return []Event{{Type: EventQuest, Details: truncate(text, 100)}}

	case "token_count":
		usage := codexUsageFromTokenInfo(msg.Info)
		if usage == nil {
			return nil
		}
		w.LastTokenUsage = usage
		return []Event{{Type: EventIdle, Details: "Context updated", TokenUsage: usage}}

	default:
		return nil
	}
}

func (w *Watcher) parseCodexResponseItem(raw json.RawMessage) []Event {
	var item codexResponseItem
	if err := json.Unmarshal(raw, &item); err != nil {
		return nil
	}

	switch item.Type {
	case "reasoning":
		return []Event{{
			Type:        EventThinking,
			Details:     "Reasoning...",
			TokenUsage:  w.LastTokenUsage,
			ThoughtText: codexReasoningText(item),
		}}

	case "message":
		text := codexMessageText(item.Content)
		if item.Role != "assistant" || text == "" {
			return nil
		}
		return []Event{{
			Type:       EventThinking,
			Details:    truncate(text, 40),
			TokenUsage: w.LastTokenUsage,
		}}

	case "function_call":
		if evt := w.parseCodexFunctionCall(item); evt != nil {
			evt.TokenUsage = w.LastTokenUsage
			return []Event{*evt}
		}

	case "function_call_output":
		return w.parseCodexFunctionCallOutput(item)

	case "web_search_call":
		return []Event{{
			Type:       EventReading,
			Details:    "Searching web",
			ToolName:   "WebSearch",
			TokenUsage: w.LastTokenUsage,
		}}
	}

	return nil
}

func (w *Watcher) parseCodexFunctionCall(item codexResponseItem) *Event {
	toolName := strings.ToLower(item.Name)
	args := decodeCodexToolArgs(item.Arguments)

	var evt *Event
	switch {
	case isCodexShellTool(toolName):
		cmd := args.Cmd
		if cmd == "" {
			cmd = args.Command
		}
		if isGitPushCommand(cmd) {
			evt = &Event{Type: EventGitPush, Details: "SHIPPED!", ToolName: displayCodexToolName(item.Name)}
		} else {
			evt = &Event{Type: EventBash, Details: codexCommandDetails(cmd), ToolName: displayCodexToolName(item.Name)}
		}

	case isCodexWriteTool(toolName):
		evt = &Event{Type: EventWriting, Details: "Writing code", ToolName: displayCodexToolName(item.Name)}

	case isCodexReadTool(toolName):
		evt = &Event{Type: EventReading, Details: "Reading context", ToolName: displayCodexToolName(item.Name)}

	case toolName == "spawn_agent":
		agentType := args.AgentType
		if agentType == "" {
			agentType = "Agent"
		}
		evt = &Event{Type: EventSpawnAgent, Details: "Agent: " + agentType, ToolName: displayCodexToolName(item.Name), ToolUseID: item.CallID}
		if item.CallID != "" {
			w.ActiveTaskAgents[item.CallID] = agentType
		}

	case toolName == "update_plan":
		evt = &Event{Type: EventTodoUpdate, Details: "Updating tasks", ToolName: displayCodexToolName(item.Name)}
		if len(args.Plan) > 0 {
			evt.TodoItems = codexPlanTodos(args.Plan)
			w.CurrentTodos = evt.TodoItems
			evt.Details = codexPlanDetails(args.Plan)
		}

	case toolName == "request_user_input":
		evt = &Event{Type: EventAskUser, Details: "Asking question", ToolName: displayCodexToolName(item.Name)}

	default:
		evt = &Event{Type: EventThinking, Details: "Using " + displayCodexToolName(item.Name), ToolName: displayCodexToolName(item.Name)}
	}

	return evt
}

func (w *Watcher) parseCodexFunctionCallOutput(item codexResponseItem) []Event {
	if item.CallID != "" {
		if agentType, ok := w.ActiveTaskAgents[item.CallID]; ok {
			delete(w.ActiveTaskAgents, item.CallID)
			return []Event{{Type: EventAgentComplete, Details: agentType, ToolUseID: item.CallID}}
		}
	}

	if !codexOutputFailed(item) {
		return nil
	}

	details := truncate(item.Output, 60)
	if details == "" {
		details = "Tool failed"
	}
	return []Event{{
		Type:    EventError,
		Details: details,
		IsError: true,
	}}
}

func codexUsageFromTokenInfo(info *codexTokenCountInfo) *TokenUsage {
	if info == nil {
		return nil
	}

	var usage TokenUsage
	if info.TotalTokenUsage != nil {
		usage = *info.TotalTokenUsage
	} else if info.LastTokenUsage != nil {
		usage = *info.LastTokenUsage
	} else {
		return nil
	}
	usage.ContextWindow = info.ModelContextWindow
	return &usage
}

func decodeCodexToolArgs(raw json.RawMessage) codexToolArgs {
	var args codexToolArgs
	if len(raw) == 0 {
		return args
	}

	var encoded string
	if err := json.Unmarshal(raw, &encoded); err == nil {
		_ = json.Unmarshal([]byte(encoded), &args)
		return args
	}

	_ = json.Unmarshal(raw, &args)
	return args
}

func isCodexShellTool(toolName string) bool {
	switch toolName {
	case "exec_command", "write_stdin", "shell", "bash":
		return true
	default:
		return false
	}
}

func isCodexWriteTool(toolName string) bool {
	return toolName == "apply_patch" ||
		strings.Contains(toolName, "patch") ||
		strings.Contains(toolName, "write") ||
		strings.Contains(toolName, "edit")
}

func isCodexReadTool(toolName string) bool {
	return strings.Contains(toolName, "read") ||
		strings.Contains(toolName, "search") ||
		strings.Contains(toolName, "find") ||
		strings.Contains(toolName, "open") ||
		strings.Contains(toolName, "list") ||
		strings.Contains(toolName, "view")
}

func displayCodexToolName(toolName string) string {
	switch toolName {
	case "exec_command":
		return "Exec"
	case "write_stdin":
		return "Process"
	case "apply_patch":
		return "Patch"
	case "update_plan":
		return "Plan"
	case "spawn_agent":
		return "Agent"
	default:
		if toolName == "" {
			return "Tool"
		}
		return toolName
	}
}

func codexCommandDetails(cmd string) string {
	if cmd == "" {
		return "Running command"
	}
	return truncate(cmd, 40)
}

func isGitPushCommand(cmd string) bool {
	return strings.Contains(strings.ToLower(cmd), "git push")
}

func codexOutputFailed(item codexResponseItem) bool {
	if strings.EqualFold(item.Status, "failed") {
		return true
	}
	if code, ok := processExitCode(item.Output); ok {
		return code != 0
	}

	lower := strings.ToLower(item.Output)
	return strings.Contains(lower, "tool call failed") ||
		strings.Contains(lower, "operation not permitted") ||
		strings.Contains(lower, "permission denied")
}

func processExitCode(output string) (int, bool) {
	const marker = "Process exited with code "
	idx := strings.Index(output, marker)
	if idx < 0 {
		return 0, false
	}

	start := idx + len(marker)
	end := start
	for end < len(output) && output[end] >= '0' && output[end] <= '9' {
		end++
	}
	if start == end {
		return 0, false
	}

	code, err := strconv.Atoi(output[start:end])
	return code, err == nil
}

func codexPlanTodos(plan []codexPlanItem) []TodoItem {
	todos := make([]TodoItem, 0, len(plan))
	for _, item := range plan {
		todos = append(todos, TodoItem{
			Content:    item.Step,
			Status:     item.Status,
			ActiveForm: item.Step,
		})
	}
	return todos
}

func codexPlanDetails(plan []codexPlanItem) string {
	completed := 0
	for _, item := range plan {
		if item.Status == "completed" {
			completed++
		}
	}
	return fmt.Sprintf("Tasks: %d/%d done", completed, len(plan))
}

func codexReasoningText(item codexResponseItem) string {
	text := codexMessageText(item.Content)
	if text != "" {
		return text
	}

	var summaries []struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(item.Summary, &summaries); err == nil {
		for _, summary := range summaries {
			if summary.Text != "" {
				return summary.Text
			}
		}
	}

	return ""
}

func codexMessageText(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}

	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text
	}

	var items []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &items); err == nil {
		var parts []string
		for _, item := range items {
			if item.Text != "" {
				parts = append(parts, item.Text)
			}
		}
		return strings.Join(parts, " ")
	}

	return ""
}
