# CLI Reference

## Server mode (default)

Start the HTTP server:

```bash
claudegql
```

Configuration is via [environment variables](/docs/reference/environment). Prints the listening addresses on startup:

```
server:     http://localhost:8765/graphql
playground: http://localhost:8765/playground
hook:       POST http://localhost:8765/hook
docs:       http://localhost:8765/docs
```

## Record mode

Read hook JSON from stdin and forward to the server:

```bash
claudegql record
```

This is the command Claude Code calls for each hook event. It silently exits on any error (network failure, server not running, etc.) so it never blocks Claude Code.

## install-hooks

Write hook configuration into `~/.claude/settings.json`:

```bash
claudegql install-hooks
```

With a custom binary path (useful after moving the binary):

```bash
claudegql install-hooks --binary /path/to/claudegql
```

Registers hooks for: `PreToolUse`, `PostToolUse`, `Stop`, `SubagentStop`, `Notification`, `SessionStart`, `SessionEnd`, `UserPromptSubmit`, `PreCompact`, `PostCompact`.

Running it again is safe — already-registered hooks are skipped.

## Make targets

| Target | Description |
|---|---|
| `make build` | Build `bin/claudegql` |
| `make run` | Build + run in foreground |
| `make run-bg` | Build + run in background (logs → `claudegql.log`) |
| `make stop` | Kill the server on port 8765 |
| `make install-hooks` | Build + register hooks in `~/.claude/settings.json` |
| `make generate` | Regenerate gqlgen code after schema changes |
| `make record-test` | Send a test hook event to the running server |
