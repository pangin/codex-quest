package main

import (
	"hash/fnv"
)

// lineDedupSet drops a JSONL line if its raw bytes have been seen recently.
// Memory is bounded by capacity via FIFO eviction (ring buffer).
//
// Why hash the whole line instead of a record field?
// Codex's event_msg payload has no stable unique id, and response_item.call_id
// is shared between paired function_call / function_call_output records — it
// is a lifecycle key, not a dedup key. Hashing the raw line is uniform across
// every record type and orthogonal to the call_id-based lifecycle tracking
// already done via Watcher.ActiveTaskAgents.
//
// claude-quest uses uuid/messageId because those fields exist there; this is
// the codex-specific adaptation of the same defense-in-depth pattern.
type lineDedupSet struct {
	seen     map[uint64]struct{}
	order    []uint64
	capacity int
	head     int
	filled   bool
}

func newLineDedupSet(capacity int) *lineDedupSet {
	return &lineDedupSet{
		seen:     make(map[uint64]struct{}, capacity),
		order:    make([]uint64, capacity),
		capacity: capacity,
	}
}

// add records line and returns true if it was newly added, false if it was a
// duplicate of a still-resident entry.
func (s *lineDedupSet) add(line string) bool {
	h := fnv.New64a()
	_, _ = h.Write([]byte(line))
	key := h.Sum64()
	if _, ok := s.seen[key]; ok {
		return false
	}
	if s.filled {
		delete(s.seen, s.order[s.head])
	}
	s.order[s.head] = key
	s.seen[key] = struct{}{}
	s.head++
	if s.head >= s.capacity {
		s.head = 0
		s.filled = true
	}
	return true
}

// reset clears the set. Codex JSONL has no compact_boundary marker today
// (claude-quest's reset trigger), so this is unused — kept ready for the day
// upstream adds one and parseCodexEventMsg routes it here.
func (s *lineDedupSet) reset() {
	for k := range s.seen {
		delete(s.seen, k)
	}
	s.head = 0
	s.filled = false
}
