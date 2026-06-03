function ago(ts) {
  if (!ts) return '—'
  const secs = Math.floor((Date.now() - new Date(ts)) / 1000)
  if (secs < 60) return `${secs}s ago`
  if (secs < 3600) return `${Math.floor(secs / 60)}m ago`
  if (secs < 86400) return `${Math.floor(secs / 3600)}h ago`
  return `${Math.floor(secs / 86400)}d ago`
}

function shortCwd(cwd) {
  if (!cwd) return ''
  return cwd.split('/').slice(-2).join('/')
}

function RateBar({ rate }) {
  const pct = Math.min(rate * 100, 100)
  const color = pct > 10 ? 'bg-red-500' : pct > 2 ? 'bg-yellow-500' : 'bg-zinc-600'
  return (
    <div className="w-20 h-1.5 bg-zinc-800 rounded-full overflow-hidden flex-shrink-0">
      <div className={`h-full ${color} rounded-full`} style={{ width: `${Math.max(pct, 2)}%` }} />
    </div>
  )
}

export default function ErrorsPanel({ sessions, toolErrorRates }) {
  // Collect all errors from all sessions, sorted newest first
  const allErrors = []
  for (const s of sessions) {
    for (const e of s.errors ?? []) {
      allErrors.push({ ...e, cwd: s.cwd, sessionId: s.id })
    }
  }
  allErrors.sort((a, b) => new Date(b.recordedAt) - new Date(a.recordedAt))

  // All tools that have been called — for the rate table
  const rates = (toolErrorRates ?? []).sort((a, b) => b.errorRate - a.errorRate)

  const hasAnyErrors = allErrors.length > 0 || rates.some(r => r.errorCount > 0)

  return (
    <div className="p-4 space-y-5">
      <div className="text-xs text-zinc-500">errors</div>

      {/* Error rates table */}
      <div className="space-y-2">
        <div className="text-xs text-zinc-600 uppercase tracking-wider">tool error rates</div>
        {rates.length === 0 ? (
          <div className="text-zinc-700 text-xs">no data yet</div>
        ) : (
          rates.map(r => {
            const pct = r.errorRate * 100
            const rateColor = pct > 10 ? 'text-red-400' : pct > 2 ? 'text-yellow-400' : 'text-zinc-500'
            return (
              <div key={r.toolName} className="space-y-0.5">
                <div className="flex items-center gap-2">
                  <span className="text-zinc-400 text-xs w-28 truncate flex-shrink-0">{r.toolName}</span>
                  <RateBar rate={r.errorRate} />
                  <span className={`text-xs tabular-nums ${rateColor}`}>
                    {pct.toFixed(1)}%
                  </span>
                  <span className="text-zinc-600 text-xs tabular-nums">
                    {r.errorCount}/{r.totalCalls >= 1000 ? `${(r.totalCalls/1000).toFixed(0)}k` : r.totalCalls}
                  </span>
                </div>
              </div>
            )
          })
        )}
      </div>

      {/* Recent error log */}
      <div className="space-y-2">
        <div className="text-xs text-zinc-600 uppercase tracking-wider">
          error log {allErrors.length > 0 && `(${allErrors.length})`}
        </div>

        {allErrors.length === 0 ? (
          <div className="text-zinc-700 text-xs py-2">
            {hasAnyErrors
              ? 'errors exist but happened outside the loaded sessions'
              : 'no errors recorded'}
          </div>
        ) : (
          <div className="space-y-2">
            {allErrors.map(e => (
              <div key={e.id} className="bg-zinc-900 border border-zinc-800 rounded px-3 py-2 space-y-1">
                <div className="flex items-center gap-2 flex-wrap">
                  <span className="text-red-400 text-xs">✕</span>
                  <span className="text-zinc-300 text-xs font-medium">{e.toolName}</span>
                  <span className="text-zinc-600 text-xs">{ago(e.recordedAt)}</span>
                </div>
                <div className="text-zinc-400 text-xs leading-relaxed break-words">
                  {e.errorMessage}
                </div>
                <div className="text-zinc-600 text-xs">{shortCwd(e.cwd)}</div>
              </div>
            ))}
          </div>
        )}
      </div>

      {!hasAnyErrors && (
        <div className="text-center py-6">
          <div className="text-zinc-600 text-xs">✓ no tool errors</div>
        </div>
      )}
    </div>
  )
}
