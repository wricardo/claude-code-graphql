# claude-code-graphql

A GraphQL analytics server that records [Claude Code](https://claude.ai/code) hook events into SQLite and exposes them via a queryable API.

## What it does

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

Open the playground: `http://localhost:8765/playground`

## What you can query

**Sessions** — every Claude Code conversation, with:
- tool usage breakdown (Bash 92×, Read 70×, …)
- skills invoked (`gqlcli`, `gitlogreview`, …)
- error count and error details
- duration, git branch, model version
- cumulative token usage (input/output/cache)
- full transcript with structured content blocks
- subagents spawned

**Projects** — aggregated view per working directory:
- session count, transcript count
- top tools and skills across all sessions

**Hooks** — raw event stream, filterable by event type, tool name, CWD, time range:
- structured `parsedInput` union (BashInput, ReadInput, EditInput, …)
- `lastAssistantMessage` and `stopHookActive` on Stop events
- cursor-based pagination via `hooksPaged`

**Search** — FTS5 full-text search across `prompt`, `tool_input`, and `tool_response`:
- returns matched hook, field name (`toolInput` / `toolResponse` / `prompt`), and snippet with `>>>match<<<` markers

**Stats** — aggregate counters:
- top tools, hooks by day, hooks by CWD, event type breakdown, tool error rates

**Transcripts** — structured `.jsonl` reading:
- typed content blocks: TextBlock, ToolUseBlock, ToolResultBlock, ThinkingBlock
- subagent transcripts

## Example queries

```graphql
# What did I work on today?
{ sessions(limit: 5) { cwd model hookCount durationSeconds skillsUsed { name count } } }

# How many tokens did a session use?
{ session(id: "…") { tokenUsage { inputTokens outputTokens cacheReadTokens cacheCreationTokens } } }

# Search all past bash commands
{ search(query: "go build") { hook { cwd recordedAt } matchField snippet } }

# What did Claude say at the end of my last session?
{ hooks(filter: { eventType: Stop }, limit: 1) { lastAssistantMessage } }

# My most-used tools overall
{ stats { topTools(limit: 10) { name count } } }
```

## Endpoints

| URL | Purpose |
|---|---|
| `http://localhost:8765/graphql` | GraphQL API |
| `http://localhost:8765/playground` | GraphQL Playground |
| `http://localhost:8765/hook` | Hook ingestion (POST) |
| `http://localhost:8765/docs` | Documentation |

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `CLAUDEGQL_PORT` | `8765` | HTTP server port |
| `CLAUDEGQL_DB` | `claudegql.db` | SQLite database path |
| `CLAUDEGQL_CLAUDE_DIR` | `~/.claude` | Claude home dir (project/transcript discovery) |
| `CLAUDEGQL_SERVER` | `http://localhost:8765` | Server URL used by `claudegql record` |

## Make targets

```bash
make build          # build bin/claudegql
make run            # build + run (foreground)
make run-bg         # build + run (background, logs to claudegql.log)
make stop           # kill server on port 8765
make install-hooks  # register hooks in ~/.claude/settings.json
make generate       # regenerate gqlgen code after schema changes
make record-test    # send a test hook event to the running server
```

## Hook events captured

`PreToolUse` · `PostToolUse` · `Stop` · `SubagentStop` · `Notification` · `SessionStart` · `SessionEnd` · `UserPromptSubmit` · `PreCompact` · `PostCompact`

## Dependencies

- Go 1.21+
- SQLite (embedded via `modernc.org/sqlite`, no system library needed)
- **LLM CLI** (optional) — any CLI that reads stdin and outputs a summary; used for the `summarizeSession` mutation. Modify `generateSummaryViaVenu()` in `graph/helpers.go` to call any CLI.
