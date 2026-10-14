# AGENTS.md

## Build Commands

```bash
go build -o cxq .
./cxq
./cxq watch ~/path/to/project
./cxq replay <jsonl-file>
./cxq doctor
go build -tags debug -o cxq . && ./cxq studio
```

## Architecture

Codex Quest is a pixel-art RPG companion that visualizes Codex operations in
real time. It watches JSONL session logs under `~/.codex/sessions` and emits
typed events consumed by the existing animation, game state, and renderer
systems.

Core files:

- `watcher.go`: generic event types, live tailing, replay mode, and watcher state.
- `codex_session.go`: Codex session discovery by `session_meta.payload.cwd`.
- `watcher_codex.go`: structured Codex JSONL parser and tool-to-event mapping.
- `main.go`: CLI parsing, doctor command, game loop, mana/context bar, XP, todos, enemies, and effects.
- `animations.go`: animation state machine.
- `renderer.go` and renderer modules: Raylib drawing, assets, UI, biomes, particles, and accessory picker.

## Codex Event Mapping

| Codex record | Event |
| --- | --- |
| `session_meta` | `EventSystemInit` |
| `event_msg` `user_message` | `EventQuest` or `EventThinkHard` |
| `event_msg` `token_count` | context bar update |
| `response_item` `reasoning` | `EventThinking` |
| shell function calls | `EventBash` |
| patch/write/edit function calls | `EventWriting` |
| web/search/read function calls | `EventReading` |
| failed function outputs | `EventError` |
| plan updates | `EventTodoUpdate` |

Keep Codex parsing structured with JSON types. Do not parse JSONL records with
regular expressions.

## Assets

- `assets/codex/spritesheet.png`
- `assets/codex/mini_spritesheet.png`
- `assets/enemies/enemy_spritesheet.png`
- `assets/ui/chest.png`

The renderer expects the Codex sprite sheets to keep the existing frame
dimensions and animation row layout.

<!-- review-cycle-rules:begin:v1 -->
## Code Review Rules

- Follow [.github/review-cycle/CONTRACT.md](.github/review-cycle/CONTRACT.md) alongside the existing repository and scoped instructions. Do not replace them.
- Review the authorized task against pinned approved documents and finite criteria. Cite source sections and actual code/test evidence; distinguish unverified checks.
- Independently assess fixes and rebuttals. Missing evidence, resolved comments, silence, and passing tests alone are not consensus.
- For `<!-- review-cycle-request:v1 -->`, provide the exact marked JSON verdict from the contract for the supplied full candidate SHA, pinned document revision, fingerprint and manifest; cite real source and candidate file lines.
- Both explicit independent verdicts and actual required tests/CI must match the same final SHA. Do not auto-merge, force-push, weaken checks or expand authority.
- Fix rounds have no count cap. Preserve duplicate/stale-event, failed/crashed attempt, stagnation and safety stops instead of uncontrolled retries.
- If task criteria, test readiness or product/architecture direction is unclear, ask the owner visibly on the PR, wait for an owner-scoped `/review-cycle` response or identified approved ADR/document update, and require fresh reviews.
<!-- review-cycle-rules:end:v1 -->
