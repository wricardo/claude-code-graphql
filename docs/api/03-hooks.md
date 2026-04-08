# Hooks API

## Queries

### hooks

List hooks with optional filtering and sorting. Returns a flat list (use `hooksPaged` for cursor pagination).

```graphql
{
  hooks(
    filter: {
      eventType: PreToolUse
      toolName: "Bash"
      sessionId: "abc-123"
      cwd: "/path/to/your/project"
      recordedAt: { from: "2026-04-01T00:00:00Z", to: "2026-04-08T00:00:00Z" }
    }
    sort: { field: recordedAt, direction: DESC }
    limit: 50
    offset: 0
  ) {
    id
    sessionId
    eventType
    toolName
    toolInput
    toolResponse
    prompt
    cwd
    recordedAt
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

**Filter fields:**

| Field | Type | Description |
|---|---|---|
| `eventType` | `EventType` | Filter by event type enum value |
| `toolName` | `String` | Filter by tool name (e.g. `"Bash"`, `"Read"`) |
| `sessionId` | `String` | Filter to a single session |
| `cwd` | `String` | Filter by working directory |
| `recordedAt` | `TimeRange` | Filter by time range (`from` / `to`) |

### hooksPaged

Cursor-based pagination for hooks. Useful for streaming large result sets.

```graphql
{
  hooksPaged(
    sessionId: "abc-123"
    eventType: "PostToolUse"
    toolName: "Bash"
    first: 20
    after: "base64cursor"
  ) {
    totalCount
    pageInfo {
      hasNextPage
      endCursor
    }
    edges {
      cursor
      node {
        id
        eventType
        toolName
        recordedAt
      }
    }
  }
}
```

The cursor encodes `recordedAt|id` as base64. Pass `pageInfo.endCursor` as `after` to fetch the next page.

### Stop event fields

`lastAssistantMessage` and `stopHookActive` are available on `Stop` and `SubagentStop` hooks:

```graphql
{
  hooks(filter: { eventType: Stop }, limit: 5) {
    sessionId
    recordedAt
    stopHookActive
    lastAssistantMessage
  }
}
```

### toolResponseFull

For large tool outputs, Claude Code writes a separate file instead of inlining the response. `toolResponseFull` reads that file:

```graphql
{
  hooks(filter: { toolName: "Bash" }, limit: 10) {
    id
    toolResponse
    toolResponseFull
  }
}
```

## Mutations

### recordHook

Manually record a hook event (used internally by `claudegql record`):

```graphql
mutation {
  recordHook(input: {
    sessionId: "abc-123"
    eventType: PreToolUse
    toolName: "Bash"
    toolInput: "{\"command\": \"ls\"}"
    cwd: "/tmp"
  }) {
    id
    recordedAt
  }
}
```
