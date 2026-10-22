import type { CoverageResponse, Summary } from '../api'
import { componentLevel, dataLevel } from '../api'
import type { Route } from '../route'

/**
 * Reference (platform-surface §4.4): where the complete detail is — every component, every table,
 * every entry point, what it relies on, the documentation. A door appears only when there is
 * evidence behind it (surface-spec §2, rule 3); it is never shown empty.
 */
export function Reference(props: { summary: Summary; coverage?: CoverageResponse; go: (r: Partial<Route>) => void }) {
  const { summary: s, coverage: cov, go } = props
  const c = s.counts ?? {}
  const firstInside = cov?.code?.find((x) => x.read && x.container && x.components > 0)?.container
  const entries = (cov?.routes ?? 0) + (cov?.pages ?? 0) + (cov?.commands ?? 0) + (cov?.jobs ?? 0)

  const doors: { title: string; what: string; count?: string; on: () => void }[] = [
    ...(firstInside
      ? [{ title: 'Every component', what: 'Each part inside an app, what it is made of and what it calls', count: c.components ? String(c.components) : undefined, on: () => go({ screen: 'explorer', level: componentLevel(firstInside) }) }]
      : []),
    ...(cov?.tables_in?.length
      ? [{ title: 'Every table', what: 'What it stores, table by table, with its columns', count: c.tables ? String(c.tables) : undefined, on: () => go({ screen: 'explorer', level: dataLevel(cov.tables_in![0]) }) }]
      : []),
    ...(entries
      ? [{ title: 'Every entry point', what: 'Every route, page, command and job — in What it does, each with what it sets off', count: String(entries), on: () => go({ screen: 'features' }) }]
      : []),
    ...(cov?.dependencies
      ? [{ title: 'What it relies on', what: 'Other people’s code, package by package, and where it is used', count: String(cov.dependencies), on: () => go({ screen: 'deps' }) }]
      : []),
    { title: 'Documentation', what: 'The arc42 chapters archdoc writes, and the ones you write beside them', on: () => go({ screen: 'docs' }) },
  ]

  return (
    <article className="reference">
      <div className="page-q">Where is the complete detail?</div>
      <h1 className="page-title">Reference</h1>
      <div className="doors">
        {doors.map((d) => (
          <button key={d.title} className="door" onClick={d.on}>
            <span className="door-title">{d.title}</span>
            {d.count && <span className="door-count">{d.count}</span>}
            <span className="door-what">{d.what}</span>
          </button>
        ))}
      </div>
    </article>
  )
}
