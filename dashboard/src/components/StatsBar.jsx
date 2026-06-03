function Sparkline({ data }) {
  if (!data || data.length === 0) return null
  const max = Math.max(...data.map(d => d.count), 1)
  return (
    <div className="flex items-end gap-1 h-9">
      {data.map(d => {
        const pct = (d.count / max) * 100
        const day = new Date(d.date + 'T12:00:00').toLocaleDateString('en', { weekday: 'short' })
        return (
          <div key={d.date} className="flex flex-col items-center gap-0.5 flex-1 h-full justify-end">
            <div
              className="w-full bg-orange-500/60 rounded-sm"
              style={{ height: `${Math.max(pct, 4)}%` }}
              title={`${d.date}: ${d.count}`}
            />
            <span className="text-zinc-600 text-[9px] leading-none">{day[0]}</span>
          </div>
        )
      })}
    </div>
  )
}

function TopTools({ tools }) {
  if (!tools || tools.length === 0) return null
  const max = tools[0]?.count ?? 1
  return (
    <div className="space-y-1">
      {tools.slice(0, 6).map(t => (
        <div key={t.name} className="flex items-center gap-2">
          <span className="text-zinc-500 text-[10px] w-14 truncate flex-shrink-0 text-right">{t.name}</span>
          <div className="flex-1 h-1 bg-zinc-800 rounded-full overflow-hidden">
            <div
              className="h-full bg-zinc-500 rounded-full"
              style={{ width: `${(t.count / max) * 100}%` }}
            />
          </div>
          <span className="text-zinc-600 text-[10px] tabular-nums w-8 flex-shrink-0">
            {t.count >= 1000 ? `${(t.count / 1000).toFixed(0)}k` : t.count}
          </span>
        </div>
      ))}
    </div>
  )
}

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
    <div className="border-b border-zinc-800 px-4 pt-3 pb-2 space-y-2">
      {/* Row 1: stat cards */}
      <div className="grid grid-cols-6 gap-2">
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

      {/* Row 2: sparkline + top tools */}
      {(stats.hooksByDay?.length > 0 || stats.topTools?.length > 0) && (
        <div className="flex gap-4 px-1">
          {stats.hooksByDay?.length > 0 && (
            <div className="flex-1">
              <div className="text-zinc-600 text-[10px] uppercase tracking-wider mb-1">7-day activity</div>
              <Sparkline data={stats.hooksByDay} />
            </div>
          )}
          {stats.topTools?.length > 0 && (
            <div className="w-56 flex-shrink-0">
              <div className="text-zinc-600 text-[10px] uppercase tracking-wider mb-1">top tools</div>
              <TopTools tools={stats.topTools} />
            </div>
          )}
        </div>
      )}
    </div>
  )
}
