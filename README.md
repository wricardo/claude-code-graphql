# claude-code-graphql

Turn Claude Code activity into a searchable local history.

`claudegql` records Claude Code hooks into SQLite, then lets you inspect your sessions from the terminal: what Claude worked on, what you asked, which files changed, which tools/skills/subagents were used, what failed, and where past context appears.

It is useful for answering questions like:

- What Claude sessions are active right now?
- What did I ask Claude to do earlier?
- Which files did a session edit?
- Which skills or subagents were used?
- What commands/tools were run most?
- Where did a feature, error, table, endpoint, or file come up before?

## Install

One-liner install from the latest GitHub release:

```bash
curl -fsSL https://raw.githubusercontent.com/wricardo/claude-code-graphql/main/install.sh | bash
```

Or build from source:

```bash
git clone https://github.com/wricardo/claude-code-graphql.git
cd claude-code-graphql
make build
sudo install -m 755 bin/claudegql /usr/local/bin/claudegql
```

## Start recording Claude Code

Start the local server:

```bash
claudegql
```

In another terminal, install Claude Code hooks:

```bash
claudegql install-hooks
```

That updates `~/.claude/settings.json` so Claude Code sends hook events to `claudegql`.

Now use Claude Code normally. New sessions, prompts, tool calls, edits, errors, and subagent events will be recorded locally.

## Basic queries

List recent sessions:

```bash
claudegql query '{ sessions(limit:5) { id status cwd model lastSeenAt hookCount } }'
```

See what a session did:

```bash
claudegql query '{ sessions(limit:3) { id prompts editedFiles toolUsage { name count } } }'
```

Find skills and subagents used:

```bash
claudegql query '{ sessions(limit:10) { id skillsUsed { name count } subagents { agentType description } } }'
```

Search prompts, tool inputs, and tool outputs:

```bash
claudegql query '{ search(query:"webhook", limit:5) { matchField snippet hook { sessionId toolName recordedAt } } }'
```

Show overall activity:

```bash
claudegql query '{ stats { totalSessions totalHooks topTools(limit:10) { name count } } }'
```

Print the compact schema when you want to explore more fields:

```bash
claudegql schema
```

## Run in the background

From a source checkout:

```bash
make run-bg         # starts on port 8765, logs to claudegql.log
make stop           # stops the server on port 8765
```

Or run the installed binary with your own process manager. By default it uses:

- server: `http://localhost:8765/graphql`
- playground: `http://localhost:8765/playground`
- database: `claudegql.db`

## CLI

| Command | Description |
|---|---|
| `claudegql` | Start the local server |
| `claudegql install-hooks [--binary PATH]` | Register Claude Code hooks |
| `claudegql query '<query>'` | Run a query and print JSON |
| `claudegql schema` | Print the compact schema |
| `claudegql record` | Hook ingestion command used by Claude Code |

## Configuration

| Variable | Default | Description |
|---|---:|---|
| `CLAUDEGQL_PORT` | `8765` | Server port |
| `CLAUDEGQL_DB` | `claudegql.db` | SQLite database path |
| `CLAUDEGQL_CLAUDE_DIR` | `~/.claude` | Claude Code directory |
| `CLAUDEGQL_SERVER` | `http://localhost:8765` | Server URL used by `record` |

## Docs

- [Session Intelligence](https://github.com/wricardo/claude-code-graphql/wiki/Session-Intelligence) — useful queries for active work, skills, subagents, edits, prompts, errors, and token usage
- [Quick Start](https://github.com/wricardo/claude-code-graphql/wiki/Quick-Start) — setup walkthrough
- [Sessions](https://github.com/wricardo/claude-code-graphql/wiki/Sessions) — session fields and examples
- [Search](https://github.com/wricardo/claude-code-graphql/wiki/Search) — full-text search examples
