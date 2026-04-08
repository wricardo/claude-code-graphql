# Quick Start

Get claude-code-graphql running and recording your Claude Code activity.

## Build

```bash
make build
```

Produces `bin/claudegql`.

## Start the server

Run in the foreground (blocks terminal):

```bash
make run
```

Or in the background (recommended — logs go to `claudegql.log`):

```bash
make run-bg
```

You should see:

```
server:     http://localhost:8765/graphql
playground: http://localhost:8765/playground
hook:       POST http://localhost:8765/hook
docs:       http://localhost:8765/docs
```

## Install hooks

Register the hooks in `~/.claude/settings.json` so Claude Code starts sending events:

```bash
make install-hooks
```

This adds `claudegql record` as the hook command for these event types:
`PreToolUse` · `PostToolUse` · `Stop` · `SubagentStop` · `Notification` · `SessionStart` · `SessionEnd` · `UserPromptSubmit` · `PreCompact` · `PostCompact`

## Verify it's working

Send a test event:

```bash
make record-test
```

Then query:

```bash
gqlcli query '{ hooks(limit: 1) { id eventType recordedAt } }'
```

Or open the Playground at `http://localhost:8765/playground` and run:

```graphql
{
  stats {
    totalSessions
    totalHooks
    hooksByEventType { eventType count }
  }
}
```

## Stop the server

```bash
make stop
```

## Next steps

- [Sessions API](/docs/api/sessions) — query your coding sessions
- [Hooks API](/docs/api/hooks) — filter and paginate raw events
- [Search](/docs/api/search) — full-text search across all hook data
- [Environment Variables](/docs/reference/environment) — configure the server
