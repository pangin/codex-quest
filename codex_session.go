package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type codexRecord struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type codexSessionMeta struct {
	ID  string `json:"id"`
	Cwd string `json:"cwd"`
}

func codexConfigDir() (string, error) {
	if dir := os.Getenv("CODEX_HOME"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex"), nil
}

func codexSessionsDir() (string, error) {
	configDir, err := codexConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "sessions"), nil
}

// FindCodexSession finds the newest rollout JSONL whose session_meta cwd matches projectDir.
func (w *Watcher) FindCodexSession(projectDir string) error {
	absPath, err := filepath.Abs(projectDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	sessionsDir, err := codexSessionsDir()
	if err != nil {
		return fmt.Errorf("failed to get Codex sessions directory: %w", err)
	}
	if _, err := os.Stat(sessionsDir); err != nil {
		return fmt.Errorf("no Codex sessions directory found\nlooked in: %s", sessionsDir)
	}

	w.ProjectRoot = filepath.Clean(absPath)
	w.ProjectDir = sessionsDir

	filePath, modTime, err := w.findNewestConversation()
	if err != nil {
		return err
	}

	w.FilePath = filePath
	w.lastModTime = modTime
	return nil
}

func (w *Watcher) findNewestCodexSessionForProject() (string, time.Time, error) {
	if w.ProjectDir == "" || w.ProjectRoot == "" {
		return "", time.Time{}, fmt.Errorf("Codex project/session paths are not set")
	}

	var newestPath string
	var newestMod time.Time

	err := filepath.WalkDir(w.ProjectDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return nil
		}
		name := d.Name()
		if !strings.HasPrefix(name, "rollout-") || !strings.HasSuffix(name, ".jsonl") {
			return nil
		}

		cwd, ok := codexSessionCwd(path)
		if !ok || !sameProjectPath(cwd, w.ProjectRoot) {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}
		if newestPath == "" || info.ModTime().After(newestMod) {
			newestPath = path
			newestMod = info.ModTime()
		}
		return nil
	})
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to scan Codex sessions: %w", err)
	}
	if newestPath == "" {
		return "", time.Time{}, fmt.Errorf("no Codex sessions found for %s\nlooked in: %s", w.ProjectRoot, w.ProjectDir)
	}

	return newestPath, newestMod, nil
}

func codexSessionCwd(path string) (string, bool) {
	file, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for i := 0; scanner.Scan() && i < 50; i++ {
		var record codexRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil || record.Type != "session_meta" {
			continue
		}

		var meta codexSessionMeta
		if err := json.Unmarshal(record.Payload, &meta); err != nil {
			return "", false
		}
		return meta.Cwd, meta.Cwd != ""
	}

	return "", false
}

func sameProjectPath(a, b string) bool {
	absA, err := filepath.Abs(a)
	if err == nil {
		a = absA
	}
	absB, err := filepath.Abs(b)
	if err == nil {
		b = absB
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

func countCodexSessionFiles(sessionsDir string) int {
	count := 0
	_ = filepath.WalkDir(sessionsDir, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasPrefix(d.Name(), "rollout-") && strings.HasSuffix(d.Name(), ".jsonl") {
			count++
		}
		return nil
	})
	return count
}

func isCodexRecordType(recordType string) bool {
	switch recordType {
	case "session_meta", "event_msg", "response_item", "turn_context":
		return true
	default:
		return false
	}
}
