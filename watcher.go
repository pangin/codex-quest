package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// codexRolloutRE matches Codex session filenames like
// rollout-2026-05-11T18-00-40-019e1644-61eb-75a0-b505-a540f0588361.jsonl.
// Verified against ~/.codex/sessions during the 2026-05-14 audit.
var codexRolloutRE = regexp.MustCompile(
	`^rollout-\d{4}-\d{2}-\d{2}T\d{2}-\d{2}-\d{2}-` +
		`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-` +
		`[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\.jsonl$`)

// activeSessionMtimeWindow is the staleness threshold for switch candidates.
// A file whose mtime is older than this is rejected as "not actively
// growing" — prevents an external touch on an old session file from luring
// the watcher away from the truly active one.
const activeSessionMtimeWindow = 30 * time.Second

// looksLikeActiveCodexSession gates file-switch candidates: must match the
// rollout filename pattern AND have been modified within the staleness window.
// The active file itself bypasses this guard (it's already accepted).
func looksLikeActiveCodexSession(path string, modTime time.Time) bool {
	if !codexRolloutRE.MatchString(filepath.Base(path)) {
		return false
	}
	return time.Since(modTime) <= activeSessionMtimeWindow
}

// EventType represents the type of Codex event.
type EventType int

const (
	EventSystemInit EventType = iota
	EventThinking
	EventReading
	EventBash
	EventWriting
	EventSuccess
	EventError
	EventIdle

	EventQuest
	EventCompact
	EventThinkHard
	EventSpawnAgent
	EventAgentComplete
	EventTodoUpdate
	EventAskUser
	EventEnemyHit
	EventVictoryPose
	EventGitPush
)

// TokenUsage tracks context window usage for the mana bar.
type TokenUsage struct {
	InputTokens         int `json:"input_tokens"`
	CacheReadTokens     int `json:"cache_read_input_tokens"`
	CacheCreationTokens int `json:"cache_creation_input_tokens"`
	OutputTokens        int `json:"output_tokens"`
	ReasoningTokens     int `json:"reasoning_output_tokens,omitempty"`
	TotalTokens         int `json:"total_tokens,omitempty"`
	ContextWindow       int `json:"-"`
}

// Total returns total tokens used.
func (t *TokenUsage) Total() int {
	if t.TotalTokens > 0 {
		return t.TotalTokens
	}
	return t.InputTokens + t.CacheReadTokens + t.CacheCreationTokens + t.OutputTokens + t.ReasoningTokens
}

// TodoItem represents a single todo item.
type TodoItem struct {
	Content    string `json:"content"`
	Status     string `json:"status"`
	ActiveForm string `json:"activeForm"`
}

// CompactInfo contains compaction metadata.
type CompactInfo struct {
	Trigger   string `json:"trigger"`
	PreTokens int    `json:"preTokens"`
}

// Event represents a parsed Codex event.
type Event struct {
	Type    EventType
	Details string

	TokenUsage  *TokenUsage
	TodoItems   []TodoItem
	CompactInfo *CompactInfo
	ToolName    string
	ToolUseID   string
	IsError     bool
	ThinkLevel  ThinkLevel
	ThoughtText string
}

// WatchMode determines how the watcher operates.
type WatchMode int

const (
	ModeLive WatchMode = iota
	ModeReplay
)

// Watcher monitors Codex sessions and emits events.
type Watcher struct {
	Events      chan Event
	Mode        WatchMode
	FilePath    string
	ProjectDir  string
	ProjectRoot string
	ReplaySpeed time.Duration
	lastPos     int64
	lastModTime time.Time

	LastTokenUsage   *TokenUsage
	CurrentTodos     []TodoItem
	ActiveTaskAgents map[string]string

	// seenLines drops any JSONL line whose raw bytes were emitted recently.
	// Bounded ring buffer; see watcher_dedup.go for rationale.
	seenLines *lineDedupSet
}

// NewWatcher creates a new event watcher.
func NewWatcher() *Watcher {
	return &Watcher{
		Events:           make(chan Event, 100),
		ReplaySpeed:      200 * time.Millisecond,
		ActiveTaskAgents: make(map[string]string),
		seenLines:        newLineDedupSet(50000),
	}
}

// FindProjectConversation finds the newest Codex session for a project directory.
func (w *Watcher) FindProjectConversation(projectDir string) error {
	return w.FindCodexSession(projectDir)
}

// findNewestConversation finds the newest matching Codex session.
func (w *Watcher) findNewestConversation() (string, time.Time, error) {
	return w.findNewestCodexSessionForProject()
}

// StartLive begins watching the session file for new events.
func (w *Watcher) StartLive() error {
	if w.FilePath == "" {
		return fmt.Errorf("no file path set, call FindProjectConversation first")
	}

	w.Mode = ModeLive

	file, err := os.Open(w.FilePath)
	if err != nil {
		return fmt.Errorf("failed to open session file: %w", err)
	}

	pos, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		file.Close()
		return fmt.Errorf("failed to seek to end: %w", err)
	}
	w.lastPos = pos
	file.Close()

	w.Events <- Event{Type: EventSystemInit, Details: "Watching: " + filepath.Base(w.FilePath)}

	go w.tailFile()
	return nil
}

// tailFile continuously watches for new lines in the file.
func (w *Watcher) tailFile() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	checkCounter := 0
	const checkInterval = 20

	for range ticker.C {
		checkCounter++
		if checkCounter >= checkInterval {
			checkCounter = 0
			if w.checkForNewerFile() {
				continue
			}
		}

		file, err := os.Open(w.FilePath)
		if err != nil {
			continue
		}

		info, err := file.Stat()
		if err != nil {
			file.Close()
			continue
		}

		if info.Size() > w.lastPos {
			if _, err := file.Seek(w.lastPos, io.SeekStart); err != nil {
				file.Close()
				continue
			}
			scanner := newJSONLScanner(file)
			for scanner.Scan() {
				for _, evt := range w.parseLine(scanner.Text()) {
					w.Events <- evt
				}
			}
			w.lastPos = info.Size()
			// Refresh baseline only on successful read so external touches
			// can't silently push lastModTime past a real switch candidate.
			w.lastModTime = info.ModTime()
		}

		file.Close()
	}
}

// checkForNewerFile checks if a newer matching Codex session exists and switches to it.
func (w *Watcher) checkForNewerFile() bool {
	if w.ProjectDir == "" || w.ProjectRoot == "" {
		return false
	}

	filePath, modTime, err := w.findNewestConversation()
	if err != nil {
		return false
	}

	if filePath == w.FilePath {
		return false
	}

	// Compare against a fresh stat of the *currently* watched file, not the
	// cached baseline — the active file is its own freshest reference and
	// must not lose to a one-off external touch on a stale sibling.
	if currentInfo, err := os.Stat(w.FilePath); err == nil {
		if !modTime.After(currentInfo.ModTime()) {
			return false
		}
	}

	// Guard against non-rollout filenames and stale candidates.
	if !looksLikeActiveCodexSession(filePath, modTime) {
		return false
	}

	// Seek the new file to EOF so only events written AFTER the switch are
	// emitted. Reading from byte 0 would re-emit the entire session and
	// double-credit XP.
	newInfo, err := os.Stat(filePath)
	if err != nil {
		return false
	}

	oldFile := filepath.Base(w.FilePath)
	newFile := filepath.Base(filePath)

	w.FilePath = filePath
	w.lastModTime = newInfo.ModTime()
	w.lastPos = newInfo.Size()

	w.Events <- Event{
		Type:    EventSystemInit,
		Details: fmt.Sprintf("Switched: %s", newFile),
	}

	fmt.Printf("Switched from %s to %s\n", oldFile, newFile)
	return true
}

// StartReplay plays through an existing Codex session file.
func (w *Watcher) StartReplay(filePath string) error {
	w.Mode = ModeReplay
	w.FilePath = filePath

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open replay file: %w", err)
	}

	w.Events <- Event{Type: EventSystemInit, Details: "Replaying: " + filepath.Base(filePath)}

	go func() {
		defer file.Close()
		scanner := newJSONLScanner(file)
		for scanner.Scan() {
			for _, evt := range w.parseLine(scanner.Text()) {
				w.Events <- evt
				time.Sleep(w.ReplaySpeed)
			}
		}
		w.Events <- Event{Type: EventSuccess, Details: "Replay complete"}
	}()

	return nil
}

func newJSONLScanner(r io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)
	return scanner
}

// parseLine parses a Codex JSONL line, deduping line-identical replays.
//
// Dedup happens here (not deeper in parseCodexLine) so it runs once per raw
// line for every record type and skips the json.Unmarshal cost on duplicates.
// Dedup set is instance-scoped: a second cxq run gets a fresh set, so user
// intent to re-watch is preserved.
func (w *Watcher) parseLine(line string) []Event {
	if w.seenLines != nil && !w.seenLines.add(line) {
		return nil
	}
	events, _ := w.parseCodexLine(line)
	return events
}

// ThinkLevel represents intensity of thinking request.
type ThinkLevel int

const (
	ThinkNone ThinkLevel = iota
	ThinkNormal
	ThinkHard
	ThinkHarder
	ThinkUltra
)

// detectThinkLevel checks user message for thinking intensity.
func detectThinkLevel(text string) ThinkLevel {
	lower := strings.ToLower(text)

	if strings.Contains(lower, "ultrathink") {
		return ThinkUltra
	}
	if strings.Contains(lower, "think harder") {
		return ThinkHarder
	}
	if strings.Contains(lower, "think hard") || strings.Contains(lower, "think deeply") ||
		strings.Contains(lower, "think carefully") || strings.Contains(lower, "deep think") {
		return ThinkHard
	}
	if strings.Contains(lower, "really think") {
		return ThinkNormal
	}

	return ThinkNone
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)

	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
