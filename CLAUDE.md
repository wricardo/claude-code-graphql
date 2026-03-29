# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run Commands

```bash
make build              # Build bin/claudegql
make generate           # Regenerate gqlgen code after schema changes
make run                # Build + run server on :8765
make install-hooks      # Register hooks in ~/.claude/settings.json
make stop               # Kill server on port 8765
make record-test        # Send a test hook event to running server
```

**Environment variables:** `CLAUDEGQL_PORT` (default 8765), `CLAUDEGQL_DB` (default claudegql.db), `CLAUDEGQL_CLAUDE_DIR` (default ~/.claude), `CLAUDEGQL_SERVER` (default http://localhost:8765).

## Architecture

GraphQL analytics server that records Claude Code hook events into SQLite and exposes them via a GraphQL API. Three layers:

1. **Hook ingestion** (`cmd/claudegql/main.go`) — HTTP handler receives hook JSON from Claude Code, CLI `record` subcommand pipes stdin to the server. Hook commands are installed into `~/.claude/settings.json` and must never block Claude Code (silent failure on errors).

2. **Store** (`internal/store/store.go`) — SQLite with FTS5. Tables: `sessions`, `hooks`, `hooks_fts` (virtual). The store handles all DB queries, error detection (parsing tool_response JSON for error patterns), skill extraction (parsing Skill tool_input for skill names), and FTS5 search. Timestamps stored by Go's time format include `+0000 UTC` suffix — use `SUBSTR(col, 1, 26)` when passing to SQLite date functions like `julianday()`.

3. **Resolvers** (`graph/schema.resolvers.go`) — GraphQL resolvers. Most Session/Project fields are computed at query time (not persisted): toolUsage, skillsUsed, errorCount, errors, durationSeconds. Summaries are generated on demand via the `summarizeSession` mutation which pipes a session digest to `venu`.

4. **Claude filesystem reader** (`internal/claude/claude.go`) — Reads `~/.claude/projects/` (project discovery), `~/.claude/skills/` (user skills), and session `.jsonl` transcripts. Project directory names encode filesystem paths by replacing `/` and `.` with `-`, which is lossy. The `resolvePathDFS()` function reconstructs paths by trying `/`, `.`, and `-` separators against the actual filesystem.

## gqlgen Workflow

Schema-first: edit `graph/schema/schema.graphqls` → run `make generate` → implement new resolver stubs in `graph/schema.resolvers.go`. The generator preserves existing resolver implementations and creates `panic("not implemented")` stubs for new fields.

Fields that need computation at query time must be marked `resolver: true` in `gqlgen.yml` under the `models` section. The `graph/generated.go` and `graph/models_gen.go` files are auto-generated — don't edit them.

## Key Patterns

- **Hook JSON flow:** Claude Code → stdin → `claudegql record` → POST `/hook` → `store.RecordHook()` (upserts session + inserts hook + triggers FTS5 indexing)
- **Error detection:** `isErrorResponse()` parses PostToolUse JSON responses looking for `exitCode != 0` or `"error"` keys. Short non-JSON responses are pattern-matched. Long responses are skipped to avoid false positives on source code containing "error" strings.
- **Cursor pagination:** Keyset using `recordedAt|id` tuple encoded as base64. Query pattern: `(recorded_at < ? OR (recorded_at = ? AND id < ?))`.
- **Session summaries:** `summarizeSession` mutation calls `Store.GetSessionActivityDigest()` (builds text with prompts, tool usage, files touched, errors) then pipes it to `venu` CLI for LLM summarization. Result cached in sessions.summary column.
- **FTS5 search:** Standalone virtual table with `hook_id UNINDEXED` column. Backfilled on startup for existing hooks. INSERT trigger keeps it in sync. Search returns hook_id + snippet with `>>>match<<<` markers.
