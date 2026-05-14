package main

import (
	"testing"
	"time"
)

func TestLineDedupSet_AddNew(t *testing.T) {
	s := newLineDedupSet(4)
	if !s.add("first") {
		t.Fatal("first add should report new")
	}
	if !s.add("second") {
		t.Fatal("distinct line should report new")
	}
}

func TestLineDedupSet_AddDuplicate(t *testing.T) {
	s := newLineDedupSet(4)
	if !s.add("same") {
		t.Fatal("first add should report new")
	}
	if s.add("same") {
		t.Fatal("second add of identical line should report duplicate")
	}
}

func TestLineDedupSet_Eviction(t *testing.T) {
	s := newLineDedupSet(4)
	// Fill capacity.
	s.add("a")
	s.add("b")
	s.add("c")
	s.add("d")
	// Capacity reached. The next insert evicts the OLDEST entry ("a"),
	// not anything younger.
	if !s.add("e") {
		t.Fatal("5th distinct line should report new")
	}
	// "b" is still resident — must still be deduped.
	// (Check this BEFORE re-adding "a", because each new add evicts one
	// more oldest entry, so re-adding "a" would also kick "b" out.)
	if s.add("b") {
		t.Fatal("still-resident entry should dedup")
	}
	// "a" was evicted — re-adding it must report new.
	if !s.add("a") {
		t.Fatal("re-add after eviction should report new")
	}
}

func TestLineDedupSet_Reset(t *testing.T) {
	s := newLineDedupSet(4)
	s.add("x")
	if s.add("x") {
		t.Fatal("pre-reset duplicate should dedup")
	}
	s.reset()
	if !s.add("x") {
		t.Fatal("post-reset add should report new")
	}
}

func TestWatcher_DedupesRepeatedLines(t *testing.T) {
	w := NewWatcher()
	line := `{"type":"event_msg","payload":{"type":"user_message","message":"hi"}}`

	first := w.parseLine(line)
	if len(first) == 0 {
		t.Fatal("first parse should emit at least one event")
	}
	second := w.parseLine(line)
	if second != nil {
		t.Fatalf("duplicate parse should return nil events, got %v", second)
	}
}

func TestWatcher_DedupSetIsInstanceScoped(t *testing.T) {
	line := `{"type":"event_msg","payload":{"type":"user_message","message":"hi"}}`

	w1 := NewWatcher()
	if got := w1.parseLine(line); len(got) == 0 {
		t.Fatal("watcher 1 should emit on first parse")
	}

	// A second Watcher instance must have its own empty dedup set so that
	// a second `cxq replay` of the same file works.
	w2 := NewWatcher()
	if got := w2.parseLine(line); len(got) == 0 {
		t.Fatal("watcher 2 should emit independently on first parse")
	}
}

func TestLooksLikeActiveCodexSession(t *testing.T) {
	const validName = "rollout-2026-05-11T18-00-40-019e1644-61eb-75a0-b505-a540f0588361.jsonl"
	now := time.Now()
	cases := []struct {
		name    string
		path    string
		modTime time.Time
		want    bool
	}{
		{"valid name, fresh", validName, now, true},
		{"valid name, just inside window", validName, now.Add(-29 * time.Second), true},
		{"valid name, stale", validName, now.Add(-2 * time.Minute), false},
		{"wrong prefix", "session-2026-05-11T18-00-40-019e1644-61eb-75a0-b505-a540f0588361.jsonl", now, false},
		{"wrong extension", "rollout-2026-05-11T18-00-40-019e1644-61eb-75a0-b505-a540f0588361.txt", now, false},
		{"missing uuid", "rollout-2026-05-11T18-00-40.jsonl", now, false},
		{"directory path is fine", "/tmp/sessions/" + validName, now, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := looksLikeActiveCodexSession(tc.path, tc.modTime); got != tc.want {
				t.Fatalf("looksLikeActiveCodexSession(%q, %v) = %v, want %v",
					tc.path, tc.modTime, got, tc.want)
			}
		})
	}
}
