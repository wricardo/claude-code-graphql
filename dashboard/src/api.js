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
    recentActivity: hooks(filter: {}, limit: 30) {
      id
      eventType
      toolName
      sessionId
      cwd
      recordedAt
      agentType
      permissionMode
    }
    recentPrompts: hooks(filter: { eventType: UserPromptSubmit }, limit: 100) {
      id
      sessionId
      prompt
      recordedAt
    }
    recentStops: hooks(filter: { eventType: Stop }, limit: 100) {
      id
      sessionId
      lastAssistantMessage
      recordedAt
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

export async function fetchSessionHistory(sessionId) {
  return gql(`
    query SessionHistory($sid: String!) {
      prompts: hooks(filter: { sessionId: $sid, eventType: UserPromptSubmit }, limit: 50) {
        id
        prompt
        recordedAt
      }
      stops: hooks(filter: { sessionId: $sid, eventType: Stop }, limit: 50) {
        id
        lastAssistantMessage
        recordedAt
      }
    }
  `, { sid: sessionId })
}
