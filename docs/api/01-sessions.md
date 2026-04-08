# Sessions API

## Queries

### sessions

List sessions ordered by most recent activity.

```graphql
{
  sessions(limit: 20, offset: 0) {
    id
    cwd
    model
    gitBranch
    hookCount
    durationSeconds
    firstSeenAt
    lastSeenAt
    toolUsage { name count }
    skillsUsed { name count }
    errorCount
    summary
    tokenUsage {
      inputTokens
      outputTokens
      cacheReadTokens
      cacheCreationTokens
    }
  }
}
```

### session

Fetch a single session by ID.

```graphql
{
  session(id: "abc-123") {
    id
    cwd
    model
    gitBranch
    errors {
      toolName
      errorMessage
      input
      recordedAt
    }
    subagents {
      id
      agentType
      description
    }
    transcript(limit: 50, offset: 0) {
      type
      role
      content
      uuid
      timestamp
      isSidechain
    }
  }
}
```

### Session.hooks

Fetch hooks within a session with the same filters as the top-level `hooks` query:

```graphql
{
  session(id: "abc-123") {
    hooks(filter: { toolName: "Bash" }, limit: 20) {
      id
      eventType
      recordedAt
      parsedInput {
        ... on BashInput { command description }
      }
    }
  }
}
```

## Mutations

### summarizeSession

Generate a 1–2 sentence LLM summary for a session The result is cached in `sessions.summary` and returned immediately.

```graphql
mutation {
  summarizeSession(sessionId: "abc-123") {
    id
    summary
  }
}
```

The server calls an LLM CLI to generate the summary. To change the command, modify `generateSummaryViaVenu()` in `graph/helpers.go`.
