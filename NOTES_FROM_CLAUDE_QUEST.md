# Notes from claude-quest

codex-quest borrows its watcher/event/XP architecture from the sibling project
[claude-quest](https://github.com/Michaelliv/claude-quest). When bugs are fixed
in claude-quest's watcher pipeline, the same code paths in codex-quest may need
audited and patched.

This file is a living changelog of upstream fixes that *might* apply here.
Anything below is a "please check, then act if relevant" note for whoever
(human or agent) is working on codex-quest next — not a guarantee that the same
bug exists in this repo.

---

## 2026-05-14 — JSONL session-switch replay + missing event dedup

### Upstream PR

`Michaelliv/claude-quest#17` — splits into two commits:

1. `fix: stop replaying historical events on JSONL session switch`
2. `feat: dedup events by uuid with compact-aware reset`

### The bug in claude-quest

When the watcher detected a different JSONL file under `~/.claude/projects/…`
had become "newest" by mtime, it would:

1. Compare the candidate's mtime against `w.lastModTime` — a baseline frozen
   at startup (or the previous switch), never refreshed while tailing. So any
   other session file whose mtime exceeded that frozen value won the
   comparison, including stale historical sessions that Claude Code
   occasionally touches for bookkeeping.
2. Set `w.lastPos = 0` on switch, then read the whole switched-to file from
   byte zero — dumping all its historical events into the channel.

The downstream effect was severe because the event pipeline has no
deduplication: every `EventReading` / `EventWriting` / `EventBash` /
`EventThinkHard` / `EventAgentComplete` etc. unconditionally calls
`Profile.RecordX()` → `AddXP(...)`. The user saw "the same context replays
sequentially while XP rockets up".

### What the fix did

In `watcher.go`:

- `tailFile()` now refreshes `w.lastModTime = info.ModTime()` after every
  successful read, so the baseline stays current.
- `checkForNewerFile()` compares the candidate's mtime against a **fresh
  `os.Stat`** of the current file, not the cached baseline. An actively-growing
  session is its own freshest reference and cannot be lost to a one-off
  external touch.
- A `looksLikeActiveSession()` guard rejects candidates whose filename isn't
  UUID-shaped or whose mtime is older than 30 s.
- On a real switch, `w.lastPos = newInfo.Size()` — seek to EOF, not offset 0.
  Only events appended **after** the switch are emitted.
- A bounded `uuidSet` (50k entries, ring-buffer with auto-eviction; ~5 MB
  hard ceiling) tracks emitted UUIDs/messageIds. `parseLine()` drops any
  line whose identity has already been seen.
- On `compact_boundary`, the set is reset (the compact line's own UUID is
  re-added so re-reading that exact line still no-ops). Post-compact rows
  in Claude Code's JSONL all have new UUIDs that only `parentUuid`-chain
  back to the compact marker, so pre-compact UUIDs are genuinely orphan
  after the boundary and the reset only frees memory at a semantically
  clean point.

### What to check in codex-quest

The codex variant watches a different format (Codex `*.jsonl` session logs
under `~/.codex/sessions`) and identity may be carried by a different field
name. Don't blindly port the patch — audit the equivalent paths first:

1. **File-switch logic.** Look at how `watcher.go` / `codex_session.go` /
   `watcher_codex.go` pick the "current" session file (probably by mtime)
   and whether they ever re-check / switch. If a switch path exists:
   - Is the baseline mtime refreshed while tailing?
   - On switch, does the watcher reset position to 0 (replay risk) or seek
     to EOF?
   - Is there a "looks like an active session" filter, or does any
     externally-touched file qualify?

2. **Event dedup.** Codex JSONL records carry their own identity (likely
   under a field like `id` on `response_item` rows, or the session's
   `session_meta.id`). Decide if a `uuidSet`-style probe at the start of
   the parser is worth adding as defense in depth. If you do, consider what
   the equivalent of a "compact boundary" is in Codex (if any) for clearing
   the set — Codex may not have compaction, in which case the ring-buffer
   eviction is the only memory bound and that's fine.

3. **XP-grant gating.** Check `progression.go` and the main event handler
   for the same "no dedup, fire-and-forget XP" pattern. If watcher fixes
   alone close the replay path, downstream gating may not be needed.

If you confirm codex-quest doesn't have any of these patterns (different
watcher design, single-file lifetime, etc.), update this note to record
that the audit was done and nothing was needed — so the next pass doesn't
re-do the work.

---

## 2026-05-14 — AUDIT COMPLETE: watcher fix ported with codex-specific deltas

### What was ported (claude-quest 1:1 mapping)
- `watcher.go::tailFile` refreshes `w.lastModTime` after each successful read
  (previously frozen at startup/switch).
- `watcher.go::checkForNewerFile` compares the candidate's mtime against a
  fresh `os.Stat` of the current file, not the cached baseline.
- New `looksLikeActiveCodexSession()` guard rejects candidates whose
  filename isn't `rollout-<ISO-ts>-<UUID>.jsonl` or whose mtime is older
  than 30 s.
- On switch, `w.lastPos = newInfo.Size()` (seek to EOF) — was `0`, which
  re-emitted every historical event in the new file and double-credited XP.

### What was adapted (NOT copy-pasted)
- **Identity**: claude-quest dedups by `uuid` / `messageId`. Codex's
  `event_msg` payload has no stable unique id, and `response_item.call_id`
  is a lifecycle pairing key (function_call ↔ function_call_output share
  it), not a dedup key. Switched to FNV-1a 64-bit hash of the raw line —
  uniform across every record type and orthogonal to the call_id-based
  lifecycle tracking in `Watcher.ActiveTaskAgents`.
- **No compact_boundary marker**: confirmed by grep — `EventCompact` has
  zero emit sites (definition-only in `watcher.go:27`, UI consumers
  elsewhere). The dedup set's `reset()` API exists but is unused; ready
  if Codex ever adds a compaction marker upstream.
- **Filename regex** matches codex's `rollout-<ISO-timestamp>-<UUID>.jsonl`,
  verified against `~/.codex/sessions` on a real machine.

### Bonus fix (out of upstream scope, same root-cause family)
- `cxq replay <file>` used to mutate the persistent profile (XP,
  SessionsStarted, etc.) — replaying the same session twice double-credited
  it. Added `GameState.ReplayMode bool`; `main()`'s replay branch sets it,
  `NewGameState(replayMode)` skips `SessionsStarted++` and the init
  `Save()`, and `HandleEvent` gates every `Profile.RecordX` /
  `Profile.Save` / `Profile.BonusChestsFound++` mutation on it.
- Floating XP indicators and flow meter still update in replay (visual
  fidelity preserved by user decision).

### Where the code lives
- Dedup type: `watcher_dedup.go` (new).
- Watcher 4-pack fix: `watcher.go` — `tailFile`, `checkForNewerFile`,
  `looksLikeActiveCodexSession`, `parseLine`.
- Replay gating: `main.go` — `GameState.ReplayMode`, `NewGameState(bool)`,
  `HandleEvent`, `main()` replay branch.
- Tests: `watcher_dedup_test.go` (new), `watcher_switch_test.go` (new),
  small Windows-TempDir fix in `watcher_codex_test.go`.

### Next audit
File-switch + dedup paths are now structurally aligned with claude-quest.
Future upstream patches should diff against this repo's `watcher.go` and
`watcher_dedup.go` specifically rather than re-deriving.
