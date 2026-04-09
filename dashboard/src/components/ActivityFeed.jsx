const EVENT_COLORS = {
  PreToolUse: 'text-yellow-400',
  PostToolUse: 'text-green-400',
  Stop: 'text-blue-400',
  SubagentStop: 'text-purple-400',
  UserPromptSubmit: 'text-orange-400',
  SessionStart: 'text-emerald-400',
  SessionEnd: 'text-zinc-400',
  Notification: 'text-cyan-400',
  PreCompact: 'text-zinc-500',
  PostCompact: 'text-zinc-500',
}

const EVENT_ICONS = {
  PreToolUse: '▶',
  PostToolUse: '✓',
  Stop: '■',
  SubagentStop: '◆',
  UserPromptSubmit: '»',
  SessionStart: '◉',
  SessionEnd: '○',
  Notification: '●',
  PreCompact: '…',
  PostCompact: '…',
}

function ago(ts) {
  const secs = Math.floor((Date.now() - new Date(ts)) / 1000)
  if (secs < 60) return `${secs}s`
  if (secs < 3600) return `${Math.floor(secs / 60)}m`
  return `${Math.floor(secs / 3600)}h`
}

function shortCwd(cwd) {
  if (!cwd) return ''
  const parts = cwd.split('/')
  return parts[parts.length - 1]
}

export default function ActivityFeed({ hooks }) {
  const sorted = [...hooks].sort(
    (a, b) => new Date(b.recordedAt) - new Date(a.recordedAt)
  )

  return (
    <div className="p-4">
      <div className="text-xs text-zinc-500 mb-3">activity</div>
      <div className="space-y-px">
        {sorted.map(hook => {
          const color = EVENT_COLORS[hook.eventType] ?? 'text-zinc-400'
          const icon = EVENT_ICONS[hook.eventType] ?? '·'
          return (
            <div
              key={hook.id}
              className="flex items-start gap-2 py-1.5 border-b border-zinc-800/50 last:border-0"
            >
              <span className={`${color} text-xs mt-0.5 w-3 flex-shrink-0`}>{icon}</span>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-1.5 flex-wrap">
                  <span className={`text-xs ${color}`}>{hook.eventType}</span>
                  {hook.toolName && (
                    <span className="text-zinc-400 text-xs">{hook.toolName}</span>
                  )}
                </div>
                <div className="flex items-center gap-2 mt-0.5 text-xs text-zinc-600">
                  {hook.cwd && <span className="truncate">{shortCwd(hook.cwd)}</span>}
                  {hook.agentType && hook.agentType !== 'claude-code' && (
                    <span className="text-purple-600">{hook.agentType}</span>
                  )}
                  <span className="flex-shrink-0">{ago(hook.recordedAt)}</span>
                </div>
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
