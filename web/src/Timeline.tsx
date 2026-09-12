import { useState } from 'react'
import type { ReactNode } from 'react'
import type { DiffResponse, Edge, Version } from './api'
import { useApi, when } from './api'
import { Failure, Loading, Prov } from './Prov'

/**
 * Every version archdoc recorded. A version exists only when the architecture changed — a run
 * that changed nothing records nothing — so each entry here is a real change.
 */
export function Timeline({ versions, onOpen }: { versions: Version[]; onOpen: (v: number) => void }) {
  const [to, setTo] = useState<number | null>(null)
  const [from, setFrom] = useState<number | null>(null)

  const sorted = [...versions].sort((a, b) => b.id - a.id)
  const target = to ?? sorted[0]?.id
  const previous = sorted.find((v) => v.id < (target ?? 0))?.id
  const base = from ?? previous

  const diff = useApi<DiffResponse>(target && base ? `/api/diff?from=${base}&to=${target}` : null)

  return (
    <div className="split">
      <section>
        <h2>Versions</h2>
        <ol className="timeline">
          {sorted.map((v) => (
            <li key={v.id} className={v.id === target ? 'active' : ''}>
              <button
                className="link"
                onClick={() => {
                  setTo(v.id)
                  setFrom(null)
                }}
              >
                Version {v.id}
              </button>
              <div className="muted small">
                {when(v.created_at)}
                {v.commit && <> · commit <span className="mono">{v.commit}</span></>}
              </div>
            </li>
          ))}
        </ol>
      </section>

      <section>
        {target && (
          <div className="toolbar">
            <h2>Version {target}</h2>
            <label>
              compared with{' '}
              <select value={base ?? ''} onChange={(e) => setFrom(Number(e.target.value))}>
                {sorted
                  .filter((v) => v.id !== target)
                  .map((v) => (
                    <option key={v.id} value={v.id}>
                      version {v.id}
                    </option>
                  ))}
              </select>
            </label>
            <button onClick={() => onOpen(target)}>Open on the diagram</button>
          </div>
        )}
        {!base && <p className="muted">This is the first version; there is nothing to compare it with.</p>}
        {diff.loading && <Loading />}
        {diff.error && <Failure error={diff.error} />}
        {diff.data && <DiffView d={diff.data} />}
      </section>
    </div>
  )
}

function DiffView({ d }: { d: DiffResponse }) {
  if (d.empty) return <p>No difference between version {d.from} and version {d.to}.</p>
  const x = d.diff
  const edge = (e: Edge) => `${e.from} → ${e.to}${e.label ? ` (${e.label})` : ''}`
  return (
    <>
      <p className={d.structural ? 'badge warn' : 'badge'}>
        {d.structural
          ? 'Structural change — elements or relationships appeared or disappeared.'
          : 'Wording only — the structure is identical.'}
      </p>
      <List title="Elements added" cls="added" items={x.added_nodes} show={(n) => <>{n.name} <Prov p={n.provenance} /></>} />
      <List title="Elements removed" cls="removed" items={x.removed_nodes} show={(n) => <>{n.name} <span className="muted small">{n.id}</span></>} />
      <List title="Relationships added" cls="added" items={x.added_edges} show={(e) => edge(e)} />
      <List title="Relationships removed" cls="removed" items={x.removed_edges} show={(e) => edge(e)} />
      {!!x.changed?.length && (
        <>
          <h3>Changed</h3>
          <table>
            <thead>
              <tr>
                <th>Element</th>
                <th>Field</th>
                <th>Before</th>
                <th>After</th>
              </tr>
            </thead>
            <tbody>
              {x.changed.map((c, i) => (
                <tr key={i}>
                  <td className="mono small">{c.element}</td>
                  <td>{c.field}</td>
                  <td className="removed">{c.before || <span className="muted">—</span>}</td>
                  <td className="added">{c.after || <span className="muted">—</span>}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </>
      )}
    </>
  )
}

function List<T>({ title, cls, items, show }: { title: string; cls: string; items: T[] | null; show: (t: T) => ReactNode }) {
  if (!items?.length) return null
  return (
    <>
      <h3>{title}</h3>
      <ul className={`plain ${cls}`}>
        {items.map((t, i) => (
          <li key={i}>{show(t)}</li>
        ))}
      </ul>
    </>
  )
}
