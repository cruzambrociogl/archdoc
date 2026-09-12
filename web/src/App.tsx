import { useState } from 'react'
import type { Summary, Version } from './api'
import { RootContext, useApi } from './api'
import { Failure, Loading } from './Prov'
import { Canvas } from './Canvas'
import { Timeline } from './Timeline'
import { Rules } from './Rules'
import { Docs } from './Docs'
import { Completeness } from './Completeness'
import { Runs } from './Runs'

const tabs = [
  ['canvas', 'Diagram'],
  ['timeline', 'History'],
  ['rules', 'Rules'],
  ['docs', 'Documents'],
  ['completeness', 'Completeness'],
  ['runs', 'Network runs'],
] as const

type Tab = (typeof tabs)[number][0]

export function App() {
  const summary = useApi<Summary>('/api/summary')
  const versions = useApi<Version[]>('/api/versions')
  const [tab, setTab] = useState<Tab>('canvas')
  const [version, setVersion] = useState<number | null>(null) // null: the latest

  if (summary.error) return <main className="page"><Failure error={summary.error} /></main>
  if (!summary.data) return <main className="page"><Loading /></main>
  const s = summary.data

  return (
    <RootContext.Provider value={s.root}>
      <header className="top">
        <div className="brand">
          archdoc <span className="repo">{s.name}</span>
          <span className="muted small">
            {s.source} · {s.commit ? `commit ${s.commit}` : 'no commit recorded'}
          </span>
        </div>
        <label className="version">
          Version{' '}
          <select
            value={version ?? ''}
            onChange={(e) => setVersion(e.target.value ? Number(e.target.value) : null)}
          >
            <option value="">latest ({s.latest_version})</option>
            {(versions.data ?? []).map((v) => (
              <option key={v.id} value={v.id}>
                {v.id} — {new Date(v.created_at).toLocaleDateString()}
                {v.commit ? ` · ${v.commit}` : ''}
              </option>
            ))}
          </select>
        </label>
      </header>

      <nav className="tabs">
        {tabs.map(([id, label]) => (
          <button key={id} className={tab === id ? 'active' : ''} onClick={() => setTab(id)}>
            {label}
            {id === 'runs' && s.runs > 0 && <span className="count">{s.runs}</span>}
          </button>
        ))}
      </nav>

      <main className="page">
        {tab === 'canvas' && <Canvas version={version} />}
        {tab === 'timeline' && (
          <Timeline
            versions={versions.data ?? []}
            onOpen={(v) => {
              setVersion(v)
              setTab('canvas')
            }}
          />
        )}
        {tab === 'rules' && <Rules />}
        {tab === 'docs' && <Docs />}
        {tab === 'completeness' && <Completeness version={version} />}
        {tab === 'runs' && <Runs />}
      </main>
    </RootContext.Provider>
  )
}
