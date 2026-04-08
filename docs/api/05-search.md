# Search API

Full-text search across all hook data using SQLite FTS5.

## Query

```graphql
{
  search(
    query: "database migration error"
    sessionId: "abc-123"
    cwd: "/path/to/your/project"
    limit: 20
  ) {
    matchField
    snippet
    hook {
      id
      sessionId
      eventType
      toolName
      cwd
      recordedAt
    }
  }
}
```

## Parameters

| Parameter | Description |
|---|---|
| `query` | FTS5 search query (required) |
| `sessionId` | Restrict search to a single session |
| `cwd` | Restrict search to a working directory |
| `limit` | Max results (default 20) |

## Response fields

| Field | Description |
|---|---|
| `matchField` | Which column matched: `prompt`, `toolInput`, or `toolResponse` |
| `snippet` | Excerpt with `>>>match<<<` markers around matched terms |
| `hook` | The full Hook object that matched |

## matchField values

- `prompt` — the match is in a UserPromptSubmit message
- `toolInput` — the match is in the tool's input (command, file path, pattern, etc.)
- `toolResponse` — the match is in the tool's output (stdout, file content, grep output, etc.)

## Examples

Search all bash commands for a pattern:

```graphql
{
  search(query: "go build") {
    matchField
    snippet
    hook { toolName cwd recordedAt }
  }
}
```

Search for errors within a project:

```graphql
{
  search(query: "exit code 1", cwd: "/path/to/your/project", limit: 10) {
    snippet
    hook { sessionId recordedAt }
  }
}
```

Find which sessions touched a specific file:

```graphql
{
  search(query: "internal/store/store.go") {
    matchField
    hook { sessionId eventType toolName }
  }
}
```
