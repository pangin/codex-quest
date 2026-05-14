package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseCodexEvents(t *testing.T) {
	w := NewWatcher()

	tests := []struct {
		name      string
		line      string
		wantType  EventType
		wantTool  string
		wantUsage int
	}{
		{
			name:     "session meta",
			line:     `{"type":"session_meta","payload":{"id":"s1","cwd":"/tmp/project"}}`,
			wantType: EventSystemInit,
		},
		{
			name:     "user quest",
			line:     `{"type":"event_msg","payload":{"type":"user_message","message":"implement this"}}`,
			wantType: EventQuest,
		},
		{
			name:      "token count",
			line:      `{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":10,"cached_input_tokens":5,"output_tokens":3,"reasoning_output_tokens":2,"total_tokens":20},"model_context_window":258400}}}`,
			wantType:  EventIdle,
			wantUsage: 20,
		},
		{
			name:     "reasoning",
			line:     `{"type":"response_item","payload":{"type":"reasoning","summary":[{"text":"checking structure"}]}}`,
			wantType: EventThinking,
		},
		{
			name:     "exec command",
			line:     `{"type":"response_item","payload":{"type":"function_call","name":"exec_command","call_id":"c1","arguments":"{\"cmd\":\"go test ./...\"}"}}`,
			wantType: EventBash,
			wantTool: "Exec",
		},
		{
			name:     "patch write",
			line:     `{"type":"response_item","payload":{"type":"function_call","name":"apply_patch","call_id":"c2","arguments":"{}"}}`,
			wantType: EventWriting,
			wantTool: "Patch",
		},
		{
			name:     "web search",
			line:     `{"type":"response_item","payload":{"type":"web_search_call","action":{"type":"search","query":"go docs"}}}`,
			wantType: EventReading,
			wantTool: "WebSearch",
		},
		{
			name:     "failed output",
			line:     `{"type":"response_item","payload":{"type":"function_call_output","call_id":"c3","output":"Process exited with code 1\nOutput:\nfailure"}}`,
			wantType: EventError,
		},
		{
			name:     "plan update",
			line:     `{"type":"response_item","payload":{"type":"function_call","name":"update_plan","call_id":"c4","arguments":"{\"plan\":[{\"step\":\"one\",\"status\":\"completed\"},{\"step\":\"two\",\"status\":\"in_progress\"}]}"}}`,
			wantType: EventTodoUpdate,
			wantTool: "Plan",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := w.parseLine(tt.line)
			if len(events) != 1 {
				t.Fatalf("got %d events, want 1", len(events))
			}
			got := events[0]
			if got.Type != tt.wantType {
				t.Fatalf("got type %v, want %v", got.Type, tt.wantType)
			}
			if tt.wantTool != "" && got.ToolName != tt.wantTool {
				t.Fatalf("got tool %q, want %q", got.ToolName, tt.wantTool)
			}
			if tt.wantUsage > 0 {
				if got.TokenUsage == nil {
					t.Fatal("expected token usage")
				}
				if got.TokenUsage.Total() != tt.wantUsage {
					t.Fatalf("got usage %d, want %d", got.TokenUsage.Total(), tt.wantUsage)
				}
			}
		})
	}
}

func TestStartReplayCodexSession(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "rollout-test.jsonl")
	// filepath.ToSlash so Windows backslashes don't become invalid JSON escapes.
	cwdJSON := filepath.ToSlash(dir)
	data := "" +
		`{"type":"session_meta","payload":{"id":"s1","cwd":"` + cwdJSON + `"}}` + "\n" +
		`{"type":"event_msg","payload":{"type":"user_message","message":"build it"}}` + "\n" +
		`{"type":"response_item","payload":{"type":"function_call","name":"apply_patch","call_id":"c1","arguments":"{}"}}` + "\n"

	if err := os.WriteFile(sessionPath, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	w := NewWatcher()
	w.ReplaySpeed = time.Millisecond
	if err := w.StartReplay(sessionPath); err != nil {
		t.Fatal(err)
	}

	var seen []EventType
	timeout := time.After(time.Second)
	for {
		select {
		case event := <-w.Events:
			seen = append(seen, event.Type)
			if event.Type == EventSuccess {
				want := []EventType{EventSystemInit, EventSystemInit, EventQuest, EventWriting, EventSuccess}
				if len(seen) != len(want) {
					t.Fatalf("got %v, want %v", seen, want)
				}
				for i := range want {
					if seen[i] != want[i] {
						t.Fatalf("got %v, want %v", seen, want)
					}
				}
				return
			}
		case <-timeout:
			t.Fatalf("timed out waiting for replay events, saw %v", seen)
		}
	}
}
