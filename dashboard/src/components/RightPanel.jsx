import { useState } from 'react'
import ActivityFeed from './ActivityFeed'
import SkillsPanel from './SkillsPanel'
import ErrorsPanel from './ErrorsPanel'

const TABS = ['activity', 'skills', 'errors']

export default function RightPanel({ hooks, sessions, userSkills, toolErrorRates }) {
  const [tab, setTab] = useState('activity')

  const totalErrors = sessions.reduce((n, s) => n + (s.errors?.length ?? 0), 0)
  const totalSkillsUsed = new Set(
    sessions.flatMap(s => (s.skillsUsed ?? []).map(sk => sk.name))
  ).size

  return (
    <div className="flex flex-col h-full border-l border-zinc-800">
      {/* Tab bar */}
      <div className="flex border-b border-zinc-800 flex-shrink-0">
        {TABS.map(t => {
          const badge =
            t === 'errors' && totalErrors > 0 ? totalErrors :
            t === 'skills' ? totalSkillsUsed :
            null
          return (
            <button
              key={t}
              onClick={() => setTab(t)}
              className={`flex-1 py-2 text-xs flex items-center justify-center gap-1.5 transition-colors ${
                tab === t
                  ? 'text-zinc-100 border-b-2 border-orange-500 -mb-px'
                  : 'text-zinc-500 hover:text-zinc-300'
              }`}
            >
              {t}
              {badge != null && (
                <span className={`text-xs px-1 rounded ${
                  t === 'errors' ? 'bg-red-900/60 text-red-400' : 'bg-zinc-800 text-zinc-400'
                }`}>
                  {badge}
                </span>
              )}
            </button>
          )
        })}
      </div>

      {/* Tab content */}
      <div className="flex-1 overflow-y-auto">
        {tab === 'activity' && <ActivityFeed hooks={hooks} />}
        {tab === 'skills' && <SkillsPanel sessions={sessions} userSkills={userSkills} />}
        {tab === 'errors' && <ErrorsPanel sessions={sessions} toolErrorRates={toolErrorRates} />}
      </div>
    </div>
  )
}
