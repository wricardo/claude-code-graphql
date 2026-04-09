import { useState, useEffect } from 'react'
import { fetchSessionHistory } from '../api'

function ago(ts) {
  if (!ts) return '—'
  const secs = Math.floor((Date.now() - new Date(ts)) / 1000)
  if (secs < 60) return `${secs}s ago`
  if (secs < 3600) return `${Math.floor(secs / 60)}m ago`
  if (secs < 86400) return `${Math.floor(secs / 3600)}h ago`
  return `${Math.floor(secs / 86400)}d ago`
}

function isActive(session) {
  if (!session.lastSeenAt) return false
  return (Date.now() - new Date(session.lastSeenAt)) < 5 * 60 * 1000
}

function shortCwd(cwd) {
  if (!cwd) return '—'
  const parts = cwd.split('/')
  return parts.slice(-2).join('/')
}

function fmtDuration(secs) {
  if (secs == null) return '—'
  if (secs < 60) return `${Math.round(secs)}s`
  if (secs < 3600) return `${Math.floor(secs / 60)}m`
  return `${Math.floor(secs / 3600)}h ${Math.floor((secs % 3600) / 60)}m`
}

function fmtTokens(usage) {
  if (!usage) return null
  const total = (usage.inputTokens ?? 0) + (usage.outputTokens ?? 0)
  if (total === 0) return null
  if (total >= 1_000_000) return `${(total / 1_000_000).toFixed(1)}M`
  if (total >= 1000) return `${(total / 1000).toFixed(0)}k`
  return String(total)
}

function truncate(text, len = 120) {
  if (!text) return null
  const clean = text.replace(/\s+/g, ' ').trim()
  return clean.length > len ? clean.slice(0, len) + '…' : clean
}

// Merge and sort prompts+stops into a conversation timeline
function buildConversation(prompts, stops) {
  const msgs = [
    ...(prompts || []).map(h => ({ type: 'user', text: h.prompt, ts: h.recordedAt, id: h.id })),
    ...(stops || []).map(h => ({ type: 'assistant', text: h.lastAssistantMessage, ts: h.recordedAt, id: h.id })),
  ].filter(m => m.text)
  return msgs.sort((a, b) => new Date(a.ts) - new Date(b.ts))
}

function ConversationHistory({ sessionId }) {
  const [history, setHistory] = useState(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    fetchSessionHistory(sessionId).then(d => {
      if (!cancelled) {
        setHistory(d)
        setLoading(false)
      }
    }).catch(() => setLoading(false))
    return () => { cancelled = true }
  }, [sessionId])

  if (loading) return <div className="text-zinc-600 text-xs py-2">loading history…</div>
  if (!history) return null

  const conversation = buildConversation(history.prompts, history.stops)

  if (conversation.length === 0) {
    return <div className="text-zinc-600 text-xs py-2">no prompts recorded yet</div>
  }

  return (
    <div className="space-y-2">
      {conversation.map(msg => (
        <div key={msg.id} className={`rounded px-3 py-2 text-xs ${
          msg.type === 'user'
            ? 'bg-zinc-800 border-l-2 border-orange-500'
            : 'bg-zinc-800/50 border-l-2 border-blue-600'
        }`}>
          <div className="flex items-center gap-2 mb-1">
            <span className={msg.type === 'user' ? 'text-orange-400' : 'text-blue-400'}>
              {msg.type === 'user' ? '▸ you' : '◆ claude'}
            </span>
            <span className="text-zinc-600">{ago(msg.ts)}</span>
          </div>
          <div className="text-zinc-300 leading-relaxed whitespace-pre-wrap break-words">
            {msg.text}
          </div>
        </div>
      ))}
    </div>
  )
}

function SessionRow({ session, recentActivity, latestPrompt, latestStop }) {
  const [open, setOpen] = useState(false)
  const active = isActive(session)
  const currentHook = recentActivity.find(h => h.sessionId === session.id)
  const tokens = fmtTokens(session.tokenUsage)
  const topTools = (session.toolUsage ?? []).slice(0, 3)

  const promptPreview = truncate(latestPrompt?.prompt)
  const stopPreview = truncate(latestStop?.lastAssistantMessage)

  return (
    <div className={`border rounded-lg mb-2 overflow-hidden transition-colors ${
      active ? 'border-green-800 bg-zinc-900' : 'border-zinc-800 bg-zinc-900/50'
    }`}>
      <button
        className="w-full text-left px-4 py-3 flex items-start gap-3 hover:bg-zinc-800/50 transition-colors"
        onClick={() => setOpen(o => !o)}
      >
        <span className={`mt-1.5 w-2 h-2 rounded-full flex-shrink-0 ${
          active ? 'bg-green-400 animate-pulse' : 'bg-zinc-600'
        }`} />

        <div className="flex-1 min-w-0">
          {/* row 1: title + badges */}
          <div className="flex items-center gap-2 flex-wrap">
            <span className="text-zinc-100 text-sm font-medium truncate">
              {shortCwd(session.cwd)}
            </span>
            {session.gitBranch && (
              <span className="text-zinc-500 text-xs">@{session.gitBranch}</span>
            )}
            {active && currentHook?.toolName && (
              <span className="text-xs bg-orange-900/50 text-orange-300 px-1.5 py-0.5 rounded">
                {currentHook.toolName}
              </span>
            )}
            {session.subagents?.length > 0 && (
              <span className="text-xs bg-purple-900/40 text-purple-300 px-1.5 py-0.5 rounded">
                {session.subagents.length} agent{session.subagents.length > 1 ? 's' : ''}
              </span>
            )}
            {session.errorCount > 0 && (
              <span className="text-xs text-red-400">{session.errorCount} err</span>
            )}
          </div>

          {/* row 2: meta */}
          <div className="flex items-center gap-3 mt-0.5 text-xs text-zinc-500 flex-wrap">
            <span>{ago(session.lastSeenAt)}</span>
            <span>{session.hookCount} hooks</span>
            {session.durationSeconds && <span>{fmtDuration(session.durationSeconds)}</span>}
            {tokens && <span>{tokens} tokens</span>}
            {session.model && <span className="truncate max-w-32">{session.model.split('-').slice(0, 3).join('-')}</span>}
            {topTools.map(t => (
              <span key={t.name} className="text-zinc-600">{t.name} ×{t.count}</span>
            ))}
          </div>

          {/* row 3: current prompt preview */}
          {promptPreview && (
            <div className="mt-1.5 text-xs text-zinc-400 leading-relaxed">
              <span className="text-orange-500 mr-1">▸</span>
              {promptPreview}
            </div>
          )}

          {/* row 4: last assistant response preview (only when collapsed) */}
          {!open && stopPreview && (
            <div className="mt-1 text-xs text-zinc-600 leading-relaxed">
              <span className="text-blue-600 mr-1">◆</span>
              {stopPreview}
            </div>
          )}
        </div>

        <span className="text-zinc-600 text-xs mt-1 flex-shrink-0">{open ? '▲' : '▼'}</span>
      </button>

      {open && (
        <div className="border-t border-zinc-800 px-4 py-3 space-y-4 text-xs">

          {/* conversation history */}
          <div>
            <div className="text-zinc-500 mb-2">conversation</div>
            <ConversationHistory sessionId={session.id} />
          </div>

          {/* tools */}
          {session.toolUsage?.length > 0 && (
            <div>
              <div className="text-zinc-500 mb-1">tools</div>
              <div className="flex flex-wrap gap-1">
                {session.toolUsage.map(t => (
                  <span key={t.name} className="bg-zinc-800 px-2 py-0.5 rounded text-zinc-300">
                    {t.name} <span className="text-zinc-500">×{t.count}</span>
                  </span>
                ))}
              </div>
            </div>
          )}

          {/* skills */}
          {session.skillsUsed?.length > 0 && (
            <div>
              <div className="text-zinc-500 mb-1">skills</div>
              <div className="flex flex-wrap gap-1">
                {session.skillsUsed.map(s => (
                  <span key={s.name} className="bg-blue-900/40 text-blue-300 px-2 py-0.5 rounded">
                    /{s.name} <span className="text-blue-600">×{s.count}</span>
                  </span>
                ))}
              </div>
            </div>
          )}

          {/* subagents */}
          {session.subagents?.length > 0 && (
            <div>
              <div className="text-zinc-500 mb-1">subagents</div>
              <div className="space-y-1">
                {session.subagents.map(a => (
                  <div key={a.id} className="flex items-start gap-2 bg-zinc-800/60 px-2 py-1.5 rounded">
                    <span className="text-purple-400">◆</span>
                    <div>
                      <span className="text-zinc-300">{a.agentType || 'agent'}</span>
                      {a.description && (
                        <div className="text-zinc-500 mt-0.5 leading-relaxed">{a.description}</div>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* token breakdown */}
          {session.tokenUsage && (
            <div>
              <div className="text-zinc-500 mb-1">tokens</div>
              <div className="flex gap-3 text-zinc-400">
                <span>in {(session.tokenUsage.inputTokens ?? 0).toLocaleString()}</span>
                <span>out {(session.tokenUsage.outputTokens ?? 0).toLocaleString()}</span>
                {session.tokenUsage.cacheReadTokens > 0 && (
                  <span className="text-zinc-600">cache-r {session.tokenUsage.cacheReadTokens.toLocaleString()}</span>
                )}
              </div>
            </div>
          )}

          {/* path + id */}
          <div className="text-zinc-600 space-y-0.5">
            <div className="break-all">{session.cwd}</div>
            <div>{session.id}</div>
          </div>
        </div>
      )}
    </div>
  )
}

export default function SessionList({ sessions, recentActivity, recentPrompts, recentStops }) {
  const sorted = [...sessions].sort(
    (a, b) => new Date(b.lastSeenAt) - new Date(a.lastSeenAt)
  )

  // index latest prompt and stop per session
  const latestPromptBySession = {}
  for (const h of (recentPrompts || [])) {
    if (!latestPromptBySession[h.sessionId] ||
        new Date(h.recordedAt) > new Date(latestPromptBySession[h.sessionId].recordedAt)) {
      latestPromptBySession[h.sessionId] = h
    }
  }
  const latestStopBySession = {}
  for (const h of (recentStops || [])) {
    if (!latestStopBySession[h.sessionId] ||
        new Date(h.recordedAt) > new Date(latestStopBySession[h.sessionId].recordedAt)) {
      latestStopBySession[h.sessionId] = h
    }
  }

  return (
    <div>
      <div className="text-xs text-zinc-500 mb-3 flex items-center gap-2">
        <span>sessions</span>
        <span className="text-zinc-700">({sessions.length})</span>
      </div>
      {sorted.map(s => (
        <SessionRow
          key={s.id}
          session={s}
          recentActivity={recentActivity}
          latestPrompt={latestPromptBySession[s.id]}
          latestStop={latestStopBySession[s.id]}
        />
      ))}
    </div>
  )
}
