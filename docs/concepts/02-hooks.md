# Hooks

Hooks are the raw events Claude Code fires during a session. Every tool call, user prompt, stop event, and notification produces a hook record.

## Hook flow

```
Claude Code fires hook → writes JSON to stdin
  → claudegql record reads stdin → POST /hook
    → hookHandler decodes payload
      → store.RecordHook(): upserts session + inserts hook + FTS5 index
```

The `claudegql record` command is designed to be silent on errors so it never blocks Claude Code.

## Hook event types

| Event | When it fires |
|---|---|
| `PreToolUse` | Before a tool executes |
| `PostToolUse` | After a tool completes successfully |
| `PostToolUseFailure` | After a tool fails |
| `UserPromptSubmit` | When you submit a message to Claude |
| `Stop` | When the main agent finishes a response |
| `SubagentStop` | When a subagent finishes |
| `Notification` | Claude Code notifications |
| `SessionStart` | Session begins |
| `SessionEnd` | Session ends |
| `PreCompact` | Before context compaction |
| `PostCompact` | After context compaction |

Additional types (`PermissionRequest`, `TaskCreated`, `TeammateIdle`, etc.) are defined in the schema for completeness but may fire in specific configurations.

## Hook fields

| Field | Description |
|---|---|
| `id` | Unique hook ID |
| `sessionId` | Parent session |
| `eventType` | One of the EventType enum values |
| `toolName` | Tool that fired (Bash, Read, Edit, …) |
| `toolInput` | Raw JSON input sent to the tool |
| `toolResponse` | Raw JSON response from the tool |
| `prompt` | User prompt text (UserPromptSubmit events) |
| `cwd` | Working directory |
| `recordedAt` | Timestamp |
| `parsedInput` | Typed union of the tool input (see below) |
| `lastAssistantMessage` | Last assistant message text (Stop/SubagentStop) |
| `stopHookActive` | Whether another stop hook was active (Stop/SubagentStop) |
| `toolResponseFull` | Full tool response from disk (for large outputs) |

## parsedInput union

`parsedInput` parses `toolInput` JSON into typed structs for common tools:

```graphql
{
  hooks(filter: { eventType: PreToolUse }, limit: 10) {
    toolName
    parsedInput {
      __typename
      ... on BashInput   { command description timeout runInBackground }
      ... on ReadInput   { filePath limit offset }
      ... on EditInput   { filePath oldString newString replaceAll }
      ... on GrepInput   { pattern path glob type outputMode }
      ... on GlobInput   { pattern path }
      ... on WriteInput  { filePath content }
    }
  }
}
```

Unknown tools return `UnknownInput { raw }`.

## Stop event fields

For `Stop` and `SubagentStop` events, Claude Code sends `last_assistant_message` and `stop_hook_active` as top-level fields. The server folds them into `tool_input` so they're accessible via the typed resolvers:

```graphql
{
  hooks(filter: { eventType: Stop }, limit: 5) {
    lastAssistantMessage
    stopHookActive
    recordedAt
  }
}
```

## Error detection

The store automatically scans `tool_response` JSON on ingestion:
- Bash responses with `exitCode != 0` → error flagged with stderr
- Responses containing explicit `"error"` keys
- Short non-JSON responses pattern-matched against known error strings
- Long responses (likely source code) are skipped to avoid false positives
