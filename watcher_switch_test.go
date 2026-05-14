package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeRolloutFile creates a UUID-named rollout JSONL whose session_meta.cwd
// matches cwd, then forces its mtime to modTime so tests can simulate
// "external touch" scenarios deterministically.
func writeRolloutFile(t *testing.T, dir, minSec, cwd string, modTime time.Time) string {
	t.Helper()
	name := fmt.Sprintf(
		"rollout-2026-05-11T18-%s-019e1644-61eb-75a0-b505-a540f0588361.jsonl",
		minSec,
	)
	path := filepath.Join(dir, name)
	body := fmt.Sprintf(`{"type":"session_meta","payload":{"id":"s%s","cwd":%q}}`+"\n", minSec, cwd)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatal(err)
	}
	return path
}

func appendToFile(t *testing.T, path, line string, modTime time.Time) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(line); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatal(err)
	}
}

func setupSwitchWatcher(t *testing.T, sessionsDir, projectRoot, currentFile string) *Watcher {
	t.Helper()
	info, err := os.Stat(currentFile)
	if err != nil {
		t.Fatal(err)
	}
	w := NewWatcher()
	w.ProjectDir = sessionsDir
	w.ProjectRoot = projectRoot
	w.FilePath = currentFile
	w.lastModTime = info.ModTime()
	w.lastPos = info.Size()
	return w
}

// drainEvents consumes any pending events until idle. Used to ignore the
// EventSystemInit emitted on a successful switch.
func drainEvents(w *Watcher, timeout time.Duration) []EventType {
	var seen []EventType
	deadline := time.After(timeout)
	for {
		select {
		case ev := <-w.Events:
			seen = append(seen, ev.Type)
		case <-deadline:
			return seen
		}
	}
}

// Regression: a stale rollout file whose mtime is bumped (e.g. external
// touch) must NOT lure the watcher away from a file that's still being
// written to.
func TestCheckForNewerFile_RejectsStaleTouch(t *testing.T) {
	sessionsDir := t.TempDir()
	projectRoot := sessionsDir // session_meta.cwd points here

	now := time.Now()
	active := writeRolloutFile(t, sessionsDir, "00-40", projectRoot, now.Add(-2*time.Second))
	// stale file with a freshly-touched mtime but no new content
	stale := writeRolloutFile(t, sessionsDir, "00-41", projectRoot, now)

	// Active file gets a real new write AFTER the stale touch — its fresh
	// mtime should win against the stale candidate.
	appendToFile(t, active,
		`{"type":"event_msg","payload":{"type":"user_message","message":"new"}}`+"\n",
		now.Add(1*time.Second))

	w := setupSwitchWatcher(t, sessionsDir, projectRoot, active)

	if w.checkForNewerFile() {
		t.Fatalf("watcher switched to stale file %q despite active file being newer", stale)
	}
	if w.FilePath != active {
		t.Fatalf("FilePath changed to %q, expected to stay on %q", w.FilePath, active)
	}
}

// Regression: a candidate whose filename isn't UUID-shaped must be rejected
// even if its mtime would otherwise qualify. (findNewestCodexSessionForProject
// only filters by "rollout-…jsonl" prefix/suffix; the active-session guard
// tightens that.)
func TestCheckForNewerFile_RejectsBadFilename(t *testing.T) {
	sessionsDir := t.TempDir()
	projectRoot := sessionsDir

	now := time.Now()
	active := writeRolloutFile(t, sessionsDir, "00-40", projectRoot, now.Add(-2*time.Second))

	// A rollout-prefixed file that doesn't match the UUID pattern.
	badName := filepath.Join(sessionsDir, "rollout-test.jsonl")
	body := fmt.Sprintf(`{"type":"session_meta","payload":{"id":"sx","cwd":%q}}`+"\n", projectRoot)
	if err := os.WriteFile(badName, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(badName, now, now); err != nil {
		t.Fatal(err)
	}

	w := setupSwitchWatcher(t, sessionsDir, projectRoot, active)

	if w.checkForNewerFile() {
		t.Fatalf("watcher switched to non-UUID file %q", badName)
	}
}

// Switching to a real newer rollout must seek to EOF so historical events
// don't get re-emitted (the XP-explosion bug from upstream).
func TestCheckForNewerFile_AcceptsAndSeeksToEOF(t *testing.T) {
	sessionsDir := t.TempDir()
	projectRoot := sessionsDir

	now := time.Now()
	active := writeRolloutFile(t, sessionsDir, "00-40", projectRoot, now.Add(-1*time.Minute))

	// Candidate is freshly written with several historical lines.
	candidate := writeRolloutFile(t, sessionsDir, "00-41", projectRoot, now)
	for i := 0; i < 5; i++ {
		appendToFile(t, candidate,
			fmt.Sprintf(`{"type":"event_msg","payload":{"type":"user_message","message":"old-%d"}}`+"\n", i),
			now)
	}

	w := setupSwitchWatcher(t, sessionsDir, projectRoot, active)

	if !w.checkForNewerFile() {
		t.Fatal("expected switch to candidate")
	}
	if w.FilePath != candidate {
		t.Fatalf("FilePath = %q, want %q", w.FilePath, candidate)
	}

	candInfo, err := os.Stat(candidate)
	if err != nil {
		t.Fatal(err)
	}
	if w.lastPos != candInfo.Size() {
		t.Fatalf("lastPos = %d, want EOF %d (would re-replay historical lines)",
			w.lastPos, candInfo.Size())
	}

	// The only event on the channel should be the system-init switch
	// notification — none of the 6 historical lines must have been read.
	seen := drainEvents(w, 100*time.Millisecond)
	for _, et := range seen {
		if et != EventSystemInit {
			t.Fatalf("unexpected event %v emitted on switch; "+
				"checkForNewerFile must not read historical events", et)
		}
	}
}
