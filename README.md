# claude-code-graphql

A GraphQL analytics server that records [Claude Code](https://claude.ai/code) hook events into SQLite and exposes them via a queryable API.

Claude Code fires hooks at key moments — before/after tool calls, when you submit a prompt, when a session ends. This server captures all of them into a local SQLite database with full-text search, then lets you query your coding activity through GraphQL.

```
Claude Code → claudegql record → POST /hook → SQLite + FTS5
                                                    ↓
          sessions · projects · hooks · transcripts · stats
```

## Quick start

```bash
make build          # build bin/claudegql
make run-bg         # start server in background (port 8765, logs → claudegql.log)
make install-hooks  # register hooks in ~/.claude/settings.json
```

Open the playground at `http://localhost:8765/playground` and start querying.

## Documentation

Full documentation is on the [GitHub Wiki](https://github.com/wricardo/claude-code-graphql/wiki):

- [Introduction](https://github.com/wricardo/claude-code-graphql/wiki/01-introduction) — What it does and why
- [Quick Start](https://github.com/wricardo/claude-code-graphql/wiki/02-quickstart) — Detailed setup
- [Sessions](https://github.com/wricardo/claude-code-graphql/wiki/Sessions) — Querying session data
- [Hooks](https://github.com/wricardo/claude-code-graphql/wiki/Hooks) — Hook events and filtering
- [Search](https://github.com/wricardo/claude-code-graphql/wiki/Search) — Full-text search
- [API Reference](https://github.com/wricardo/claude-code-graphql/wiki/API-Sessions) — All queries and mutations
