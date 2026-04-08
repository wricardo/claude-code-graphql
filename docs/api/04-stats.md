# Stats API

Aggregate statistics across all recorded data.

## Query

```graphql
{
  stats {
    totalSessions
    totalHooks
    totalErrors
    avgHooksPerSession
    hooksByEventType { eventType count }
    topTools(limit: 10) { name count }
    hooksByDay(days: 30) { date count }
    hooksByCwd(limit: 10) { cwd hookCount sessionCount }
    toolErrorRates {
      toolName
      totalCalls
      errorCount
      errorRate
    }
  }
}
```

## Fields

| Field | Description |
|---|---|
| `totalSessions` | Total sessions in the database |
| `totalHooks` | Total hook events recorded |
| `totalErrors` | Total detected tool errors |
| `avgHooksPerSession` | Average hooks per session |
| `hooksByEventType` | Hook count per event type |
| `topTools(limit)` | Most-used tools by call count |
| `hooksByDay(days)` | Hook counts grouped by calendar day for the last N days (sparse — days with zero hooks are omitted) |
| `hooksByCwd(limit)` | Most active working directories by hook count |
| `toolErrorRates` | Per-tool error rate: total calls, error count, and ratio |

## Example: activity heatmap data

```graphql
{ stats { hooksByDay(days: 7) { date count } } }
```

## Example: which tools fail most often

```graphql
{
  stats {
    toolErrorRates {
      toolName
      totalCalls
      errorCount
      errorRate
    }
  }
}
```
