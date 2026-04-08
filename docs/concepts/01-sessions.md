# Sessions

A session represents a single Claude Code conversation. Sessions are automatically created on first hook receipt and updated as hooks arrive.

## Session fields

Most fields are computed at query time from the underlying hook rows, not persisted separately:

| Field | Type | Description |
|---|---|---|
| `id` | `ID!` | Claude Code session ID |
| `cwd` | `String` | Working directory of the session |
| `firstSeenAt` | `Time!` | Timestamp of the first hook event |
| `lastSeenAt` | `Time!` | Timestamp of the most recent hook event |
| `durationSeconds` | `Float` | Seconds between first and last hook |
| `hookCount` | `Int!` | Total hooks recorded in this session |
| `model` | `String` | Claude model used (e.g. `claude-sonnet-4-6`) |
| `gitBranch` | `String` | Git branch active during the session |
| `toolUsage` | `[ToolStat!]!` | Per-tool call counts, sorted by frequency |
| `skillsUsed` | `[SkillUsage!]!` | Skills invoked via the Skill tool |
| `errorCount` | `Int!` | Number of detected tool errors |
| `errors` | `[ToolError!]!` | Detailed error events |
| `tokenUsage` | `TokenUsage` | Cumulative token counts from transcript |
| `summary` | `String` | Cached LLM-generated summary |
| `hooks` | `[Hook!]!` | Raw hook events (supports same filters as top-level `hooks`) |
| `transcript` | `[TranscriptMessage!]!` | Structured `.jsonl` transcript messages |
| `subagents` | `[Subagent!]!` | Subagent sessions spawned during this session |

## Token usage

Token counts are read from the session's `.jsonl` transcript file, not from hooks. Fields:

```graphql
tokenUsage {
  inputTokens
  outputTokens
  cacheReadTokens
  cacheCreationTokens
}
```

## Session summaries

The `summarizeSession` mutation builds a text digest (prompts, tools, files touched, errors) and pipes it to an LLM CLI tool for summarization. The result is cached in `sessions.summary`.

```graphql
mutation {
  summarizeSession(sessionId: "abc-123") {
    id
    summary
  }
}
```

## Querying sessions

List sessions most-recently-active first:

```graphql
{
  sessions(limit: 10, offset: 0) {
    id
    cwd
    model
    gitBranch
    hookCount
    durationSeconds
    toolUsage { name count }
    skillsUsed { name count }
    errorCount
    tokenUsage { inputTokens outputTokens cacheReadTokens cacheCreationTokens }
  }
}
```

Get a single session by ID:

```graphql
{
  session(id: "abc-123") {
    id
    cwd
    summary
    errors { toolName errorMessage input recordedAt }
  }
}
```
