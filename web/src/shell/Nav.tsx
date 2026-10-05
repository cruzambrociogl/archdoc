import type { CoverageResponse, SavedView, Summary } from '../api'
import type { Route, Screen } from '../route'

interface Item {
  label: string
  screen?: Screen
  level?: string
  view?: SavedView
  depth: 0 | 1 | 2
  badge?: string
  header?: boolean
  divider?: boolean
}

/**
 * The left navigation. A view appears only when there is evidence behind it (surface-spec §2,
 * rule 3): components, features, flows, the data model and dependencies arrive with the code
 * lenses, and are absent until then rather than shown empty.
 */
export function Nav(props: { summary: Summary; route: Route; go: (r: Partial<Route>) => void; open: boolean; views: SavedView[]; coverage?: CoverageResponse }) {
  const { summary: s, route, go } = props
  const inExplorer = route.screen === 'explorer'
  const level = route.level ?? 'container'

  const items: Item[] = [
    { label: 'Overview', screen: 'overview', depth: 0 },
    { label: 'Architecture', depth: 0, header: true },
    { label: 'Explorer', screen: 'explorer', depth: 1, badge: '⤢' },
    ...(inExplorer
      ? ([
          { label: 'Context', screen: 'explorer', level: 'context', depth: 2 },
          { label: 'Containers', screen: 'explorer', level: 'container', depth: 2 },
        ] as Item[])
      : []),
    ...(props.views.length
      ? ([{ label: 'Saved views', depth: 1, header: true }, ...props.views.map((v) => ({ label: v.name, view: v, depth: 2, badge: 'view' }))] as Item[])
      : []),
    { label: 'Changes', screen: 'changes', depth: 0, badge: s.versions > 1 ? `${s.versions} versions` : undefined },
    { label: 'Documents', screen: 'docs', depth: 0 },
    // Earned: present once generate has written the coverage report.
    ...(props.coverage
      ? ([{ label: 'Coverage', screen: 'coverage', depth: 0, badge: `${props.coverage.gaps.reduce((n, g) => n + g.gaps.length, 0)} gaps` }] as Item[])
      : []),
    { label: 'Corrections', screen: 'rules', depth: 0, divider: true },
    { label: 'Network runs', screen: 'runs', depth: 0, badge: String(s.runs) },
  ]

  return (
    <nav className={`nav ${props.open ? 'open' : ''}`} aria-label="Sections">
      {items.map((it, i) => {
        const active = it.view
          ? inExplorer && level === it.view.level && (route.focus ?? '') === (it.view.focus ?? '') && (route.q ?? '') === (it.view.find ?? '') && (route.dim === '1') === !!it.view.dim
          : it.level
            ? inExplorer && level === it.level
            : it.screen === route.screen
        return (
          <div key={i}>
            {it.divider && <div className="nav-divider" />}
            {it.header ? (
              <div className={`nav-item nav-header depth-${it.depth}`}>{it.label}</div>
            ) : (
              <button
                className={`nav-item depth-${it.depth} ${active ? 'active' : ''}`}
                onClick={() =>
                  go(
                    it.view
                      ? { screen: 'explorer', level: it.view.level, focus: it.view.focus, q: it.view.find, dim: it.view.dim ? '1' : undefined }
                      : it.level
                        ? { screen: 'explorer', level: it.level }
                        : { screen: it.screen! },
                  )
                }
              >
                <span className="nav-label">{it.label}</span>
                {it.badge && <span className="nav-badge">{it.badge}</span>}
              </button>
            )}
          </div>
        )
      })}
    </nav>
  )
}
