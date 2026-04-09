import { useState, useEffect, useCallback } from 'react'
import { fetchDashboard } from './api'
import StatsBar from './components/StatsBar'
import SessionList from './components/SessionList'
import ActivityFeed from './components/ActivityFeed'

const POLL_MS = 5000

export default function App() {
  const [data, setData] = useState(null)
  const [error, setError] = useState(null)
  const [lastUpdated, setLastUpdated] = useState(null)

  const load = useCallback(async () => {
    try {
      const d = await fetchDashboard()
      setData(d)
      setLastUpdated(new Date())
      setError(null)
    } catch (e) {
      setError(e.message)
    }
  }, [])

  useEffect(() => {
    load()
    const id = setInterval(load, POLL_MS)
    return () => clearInterval(id)
  }, [load])

  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-100 font-mono">
      <header className="border-b border-zinc-800 px-6 py-3 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <span className="text-orange-400 font-bold tracking-tight text-lg">Claude Code</span>
          <span className="text-zinc-500 text-sm">dashboard</span>
        </div>
        <div className="flex items-center gap-4 text-xs text-zinc-500">
          {error && <span className="text-red-400">⚠ {error}</span>}
          {lastUpdated && (
            <span>updated {lastUpdated.toLocaleTimeString()}</span>
          )}
          <span className="flex items-center gap-1">
            <span className="w-2 h-2 rounded-full bg-green-500 animate-pulse inline-block" />
            polling {POLL_MS / 1000}s
          </span>
        </div>
      </header>

      {data && (
        <StatsBar stats={data.stats} sessions={data.sessions} />
      )}

      <div className="flex h-[calc(100vh-8.5rem)] overflow-hidden">
        <main className="flex-1 overflow-y-auto p-4">
          {data ? (
            <SessionList
              sessions={data.sessions}
              recentActivity={data.recentActivity}
              recentPrompts={data.recentPrompts}
              recentStops={data.recentStops}
            />
          ) : !error ? (
            <div className="text-zinc-500 text-sm mt-8 text-center">connecting…</div>
          ) : null}
        </main>

        <aside className="w-80 border-l border-zinc-800 overflow-y-auto">
          {data && <ActivityFeed hooks={data.recentActivity} />}
        </aside>
      </div>
    </div>
  )
}
