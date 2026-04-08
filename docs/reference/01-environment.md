# Environment Variables

Configuration via environment variables.

| Variable | Default | Description |
|---|---|---|
| `CLAUDEGQL_PORT` | `8765` | HTTP server port |
| `CLAUDEGQL_DB` | `claudegql.db` | SQLite database file path |
| `CLAUDEGQL_CLAUDE_DIR` | `~/.claude` | Claude Code home directory (for project discovery and session reading) |
| `CLAUDEGQL_SERVER` | `http://localhost:8765` | Server URL used by `claudegql record` subcommand |

## Examples

Run on a different port:

```bash
CLAUDEGQL_PORT=9000 claudegql
```

Use a custom database location:

```bash
CLAUDEGQL_DB=/tmp/claudegql.db claudegql
```

Point to a non-standard Claude directory:

```bash
CLAUDEGQL_CLAUDE_DIR=/custom/claude/dir claudegql
```
