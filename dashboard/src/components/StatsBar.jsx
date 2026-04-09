export default function StatsBar({ stats, sessions }) {
  const activeSessions = sessions.filter(s => {
    if (!s.lastSeenAt) return false
    return (Date.now() - new Date(s.lastSeenAt)) < 5 * 60 * 1000
  })

  const totalSubagents = sessions.reduce((n, s) => n + (s.subagents?.length ?? 0), 0)
  const totalErrors = sessions.reduce((n, s) => n + (s.errorCount ?? 0), 0)

  const cards = [
    { label: 'sessions', value: stats.totalSessions },
    { label: 'active (5m)', value: activeSessions.length, highlight: activeSessions.length > 0 },
    { label: 'hooks', value: stats.totalHooks.toLocaleString() },
    { label: 'subagents', value: totalSubagents },
    { label: 'errors', value: totalErrors, dim: totalErrors === 0, warn: totalErrors > 0 },
    { label: 'avg hooks/session', value: stats.avgHooksPerSession.toFixed(1) },
  ]

  return (
    <div className="border-b border-zinc-800 px-4 py-3 grid grid-cols-6 gap-2">
      {cards.map(c => (
        <div key={c.label} className="bg-zinc-900 rounded px-3 py-2">
          <div className={`text-xl font-bold tabular-nums ${
            c.highlight ? 'text-green-400' :
            c.warn ? 'text-red-400' :
            c.dim ? 'text-zinc-600' :
            'text-zinc-100'
          }`}>
            {c.value}
          </div>
          <div className="text-zinc-500 text-xs mt-0.5">{c.label}</div>
        </div>
      ))}
    </div>
  )
}
