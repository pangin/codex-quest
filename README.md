# Codex Quest

Pixel-art RPG companion for Codex sessions.

Codex Quest watches local Codex JSONL session logs and turns agent activity into
animations: reasoning, shell commands, file edits, web searches, tool failures,
plan updates, and context usage all become visible in a small Raylib window.

```bash
npm install -g codex-quest
```

Run it in a second terminal from the same project directory where Codex is
working:

```bash
cxq
```

## Usage

```bash
cxq                     # Watch the current project's newest Codex session
cxq watch ~/dir         # Watch a specific project's newest Codex session
cxq replay <file.jsonl> # Replay an existing Codex session
cxq doctor              # Check paths, assets, and session JSONL structure
```

Codex Quest reads sessions from:

```text
~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl
```

The live watcher selects the newest session whose `session_meta.payload.cwd`
matches the project being watched.

## Development

```bash
go build -o cxq .
./cxq doctor
./cxq replay ~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl -s 20
```

Studio mode is available in debug builds:

```bash
go build -tags debug -o cxq . && ./cxq studio
```

Requires Go 1.21+ and CGO because Raylib uses C bindings.
