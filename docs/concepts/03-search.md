# FTS5 Search

claude-code-graphql uses SQLite's FTS5 extension for full-text search across all hook data.

## What gets indexed

Three columns are indexed per hook:

| Column | Content |
|---|---|
| `prompt` | User prompt text (UserPromptSubmit events) |
| `tool_input` | JSON input sent to the tool |
| `tool_response` | JSON response from the tool (stdout, stderr, file content, etc.) |

The `hook_id` column is stored but not indexed (`UNINDEXED`), used to join results back to the hooks table.

## How sync works

An `AFTER INSERT` trigger on the `hooks` table keeps the FTS5 index up to date automatically. On server startup, any existing hooks not yet indexed are backfilled.

## Match field detection

Search results include a `matchField` indicating which column matched:

- `prompt` — the match was in the user's message text
- `toolInput` — the match was in a tool's input (command, file path, pattern, etc.)
- `toolResponse` — the match was in a tool's output (stdout, file content, grep results, etc.)

This is determined by running per-column `snippet()` calls and checking which one contains highlight markers.

## Snippet format

Snippets include `>>>` and `<<<` markers around matched terms:

```
"...ran >>>go build<<< and got compilation errors..."
```

## Limitations

- FTS5 tokenization: partial word matches may not work (search `build`, not `buil`)
- Searches are case-insensitive
- Very long tool responses (source code files) are indexed in full — use specific queries to avoid noise
