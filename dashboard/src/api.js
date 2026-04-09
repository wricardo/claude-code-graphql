const ENDPOINT = '/graphql'

async function gql(query, variables = {}) {
  const res = await fetch(ENDPOINT, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ query, variables }),
  })
  const json = await res.json()
  if (json.errors) throw new Error(json.errors[0].message)
  return json.data
}

export async function fetchDashboard() {
  return gql(`{
    sessions(limit: 50) {
      id
      cwd
      model
      gitBranch
      hookCount
      errorCount
      durationSeconds
      firstSeenAt
      lastSeenAt
      summary
      toolUsage { name count }
      skillsUsed { name count }
      subagents { id agentType description }
      tokenUsage { inputTokens outputTokens cacheReadTokens cacheCreationTokens }
    }
    hooks(filter: {}, limit: 30) {
      id
      eventType
      toolName
      sessionId
      cwd
      recordedAt
      agentType
      permissionMode
    }
    stats {
      totalSessions
      totalHooks
      totalErrors
      avgHooksPerSession
      topTools(limit: 8) { name count }
      hooksByDay(days: 7) { date count }
    }
  }`)
}
