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
