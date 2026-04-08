# Introduction

claude-code-graphql is a GraphQL analytics server that records Claude Code hook events into SQLite and exposes them via a GraphQL API.

## What it does

When you use Claude Code, it fires hooks at key moments — before and after tool calls, when you submit a prompt, when a session ends, and more. claude-code-graphql captures these events, stores them in a local SQLite database with full-text search, and lets you query your coding activity through GraphQL.

## Key features

- **Hook ingestion** — HTTP endpoint receives hook JSON from Claude Code via `claudegql record`
- **SQLite + FTS5** — persistent storage with full-text search across prompt, tool input, and tool response
- **Session analytics** — tool usage, skill tracking, error detection, duration, git branch, model, token usage
- **Transcript reading** — reads `.jsonl` session files for structured message content and subagents
- **Project discovery** — reads `~/.claude/projects/` to surface per-project activity
- **Structured tool inputs** — `parsedInput` union gives typed access to Bash, Read, Edit, Grep, Glob, Write inputs
- **Stop event fields** — `lastAssistantMessage` and `stopHookActive` parsed from Stop/SubagentStop events
- **Session summaries** — LLM-powered summaries via an LLM CLI tool
- **Aggregate stats** — top tools, hooks by day, error rates, activity by CWD

## How it works

```
Claude Code → stdin → claudegql record → POST /hook → store.RecordHook()
                                                            ↓
                                              upserts session row
                                              inserts hook row
                                              triggers FTS5 indexing
                                                            ↓
GraphQL ← sessions · projects · hooks · transcripts · stats · search
```

## The platform at a glance

```
http://localhost:8765/graphql      → GraphQL API
http://localhost:8765/playground   → GraphQL Playground
http://localhost:8765/hook         → Hook ingestion endpoint (POST)
http://localhost:8765/docs         → Documentation
```

## Documentation map

**Getting started:**
- [Quick Start](/docs/quickstart) — build, run, and record your first hook

**Concepts:**
- [Sessions](/docs/concepts/sessions) — how sessions are tracked and queried
- [Hooks](/docs/concepts/hooks) — the hook event system and event types
- [FTS5 Search](/docs/concepts/search) — full-text search across hook data

**API reference:**
- [Sessions](/docs/api/sessions) — session queries and summarization
- [Projects](/docs/api/projects) — project-level analytics
- [Hooks](/docs/api/hooks) — raw hook queries with filtering and pagination
- [Stats](/docs/api/stats) — aggregate statistics
- [Search](/docs/api/search) — full-text search
- [Transcripts](/docs/api/transcripts) — reading session transcripts

**Reference:**
- [Environment Variables](/docs/reference/environment) — configuration options
- [CLI](/docs/reference/cli) — command reference
