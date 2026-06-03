function ago(ts) {
  if (!ts) return null
  const secs = Math.floor((Date.now() - new Date(ts)) / 1000)
  if (secs < 3600) return `${Math.floor(secs / 60)}m ago`
  if (secs < 86400) return `${Math.floor(secs / 3600)}h ago`
  return `${Math.floor(secs / 86400)}d ago`
}

function Bar({ pct, color = 'bg-orange-500' }) {
  return (
    <div className="w-24 h-1.5 bg-zinc-800 rounded-full overflow-hidden flex-shrink-0">
      <div className={`h-full ${color} rounded-full`} style={{ width: `${Math.max(pct, 2)}%` }} />
    </div>
  )
}

export default function SkillsPanel({ sessions, userSkills }) {
  // Aggregate skill usage across all sessions
  const skillTotals = {}   // name → total count
  const skillLastSeen = {} // name → most recent session.lastSeenAt

  for (const session of sessions) {
    for (const sk of session.skillsUsed ?? []) {
      skillTotals[sk.name] = (skillTotals[sk.name] ?? 0) + sk.count
      if (!skillLastSeen[sk.name] || session.lastSeenAt > skillLastSeen[sk.name]) {
        skillLastSeen[sk.name] = session.lastSeenAt
      }
    }
  }

  // Build the full skill list: installed skills + any "ghost" skills seen in sessions
  const installedNames = new Set((userSkills ?? []).map(s => s.name))
  const allUsedNames = new Set(Object.keys(skillTotals))

  // Ghost skills: used in sessions but not in userSkills anymore
  const ghostNames = [...allUsedNames].filter(n => !installedNames.has(n))

  const maxCount = Math.max(...Object.values(skillTotals), 1)

  const usedInstalled = (userSkills ?? [])
    .filter(s => skillTotals[s.name] > 0)
    .sort((a, b) => (skillTotals[b.name] ?? 0) - (skillTotals[a.name] ?? 0))

  const unusedInstalled = (userSkills ?? [])
    .filter(s => !skillTotals[s.name])
    .sort((a, b) => a.name.localeCompare(b.name))

  const ghosts = ghostNames
    .sort((a, b) => (skillTotals[b] ?? 0) - (skillTotals[a] ?? 0))

  return (
    <div className="p-4 space-y-5">
      <div className="text-xs text-zinc-500">skills</div>

      {/* Used skills */}
      {usedInstalled.length > 0 && (
        <div className="space-y-3">
          <div className="text-xs text-zinc-600 uppercase tracking-wider">active ({usedInstalled.length})</div>
          {usedInstalled.map(sk => {
            const count = skillTotals[sk.name] ?? 0
            const last = skillLastSeen[sk.name]
            const pct = (count / maxCount) * 100
            return (
              <div key={sk.name} className="space-y-1">
                <div className="flex items-center gap-2">
                  <span className="text-orange-400 text-xs w-3 flex-shrink-0">◆</span>
                  <span className="text-zinc-200 text-xs font-medium">/{sk.name}</span>
                  <Bar pct={pct} />
                  <span className="text-zinc-400 text-xs tabular-nums">{count}×</span>
                  {last && <span className="text-zinc-600 text-xs">{ago(last)}</span>}
                </div>
                {sk.description && (
                  <div className="ml-5 text-zinc-500 text-xs leading-relaxed line-clamp-2">
                    {sk.description}
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}

      {/* Ghost skills — used in sessions but no longer installed */}
      {ghosts.length > 0 && (
        <div className="space-y-2">
          <div className="text-xs text-zinc-600 uppercase tracking-wider">
            ghost — used but not installed ({ghosts.length})
          </div>
          {ghosts.map(name => {
            const count = skillTotals[name] ?? 0
            const last = skillLastSeen[name]
            const pct = (count / maxCount) * 100
            return (
              <div key={name} className="flex items-center gap-2">
                <span className="text-yellow-700 text-xs w-3 flex-shrink-0">◇</span>
                <span className="text-zinc-500 text-xs">/{name}</span>
                <Bar pct={pct} color="bg-yellow-800" />
                <span className="text-zinc-600 text-xs tabular-nums">{count}×</span>
                {last && <span className="text-zinc-700 text-xs">{ago(last)}</span>}
              </div>
            )
          })}
        </div>
      )}

      {/* Unused installed skills */}
      {unusedInstalled.length > 0 && (
        <div className="space-y-2">
          <div className="text-xs text-zinc-600 uppercase tracking-wider">
            installed, never used ({unusedInstalled.length})
          </div>
          {unusedInstalled.map(sk => (
            <div key={sk.name} className="space-y-0.5">
              <div className="flex items-center gap-2">
                <span className="text-zinc-700 text-xs w-3 flex-shrink-0">·</span>
                <span className="text-zinc-600 text-xs">/{sk.name}</span>
              </div>
              {sk.description && (
                <div className="ml-5 text-zinc-700 text-xs leading-relaxed line-clamp-1">
                  {sk.description}
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
