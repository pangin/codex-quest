# Codex Quest Plan

## Goal

Build `codex-quest` as a Codex-focused pixel RPG companion that watches local
Codex session logs and visualizes agent activity in real time.

## Repository Strategy

Use `pangin/codex-quest` as the canonical repository.

Recommended remote:

```bash
origin git@github.com:pangin/codex-quest.git
```

## Codex Port Scope

Keep the existing Go/Raylib renderer and game loop architecture. The main port
is the event source:

```text
~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl
```

Live mode selects the newest rollout whose `session_meta.payload.cwd` matches
the project being watched.

## Codex Event Mapping

| Codex record | Target event |
| --- | --- |
| `session_meta` | `EventSystemInit` |
| `event_msg` with `user_message` | `EventQuest` |
| `event_msg` with `token_count` | update context bar |
| `response_item` with `reasoning` | `EventThinking` |
| `response_item` with shell/exec function call | `EventBash` |
| `response_item` with patch/write/edit function call | `EventWriting` |
| `response_item` with web/search/read function call | `EventReading` |
| `response_item` with failed function output | `EventError` |
| `update_plan` function call | `EventTodoUpdate` |

Keep the parser structured with JSON types. Do not regex JSONL records.

## File Layout

```text
watcher.go
codex_session.go
watcher_codex.go
brand.go
assets/codex/spritesheet.png
assets/codex/mini_spritesheet.png
```

## Validation Checklist

1. Build the imported app before port changes.
2. Add Codex replay mode against a saved `~/.codex/sessions/.../rollout-*.jsonl`.
3. Verify live mode selects the newest session for the active cwd.
4. Verify animations trigger for command, edit, search, reasoning, error, plan,
   and token-count updates.
5. Confirm package metadata, release config, docs, preferences, and profile
   paths use Codex Quest naming.
