import type { CoverageResponse, SavedView, Summary } from '../api'
import { published } from '../api'
import { planned } from '../planned'
import { componentLevel, dataLevel } from '../api'
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
  /** Designed, not built: drawn dashed, and opens the list of what is still to build. */
  planned?: string
}

/**
 * The left navigation. A view appears only when there is evidence behind it (surface-spec §2,
 * rule 3): components appear once some container's code was read; features, flows, the data model
 * and dependencies arrive with their lenses, and are absent until then rather than shown empty.
 */
export function Nav(props: { summary: Summary; route: Route; go: (r: Partial<Route>) => void; open: boolean; views: SavedView[]; coverage?: CoverageResponse }) {
  const { summary: s, route, go } = props
  const inExplorer = route.screen === 'explorer'
  const level = route.level ?? 'context'
  // Earned: a container whose code was read opens onto its components.
  const firstInside = props.coverage?.code?.find((c) => c.read && c.container && c.components > 0)?.container

  const items: Item[] = [
    { label: 'Overview', screen: 'overview', depth: 0 },
    { label: 'Architecture', depth: 0, header: true },
    { label: 'Explorer', screen: 'explorer', depth: 1, badge: '⤢' },
    ...(inExplorer
      ? ([
          { label: 'Context', screen: 'explorer', level: 'context', depth: 2 },
          ...(s.tiny ? [] : [{ label: 'Containers', screen: 'explorer', level: 'container', depth: 2 } as Item]),
          ...(firstInside ? [{ label: 'Components', screen: 'explorer', level: componentLevel(firstInside), depth: 2 }] : []),
          ...(props.coverage?.tables_in?.length ? [{ label: 'Data', screen: 'explorer', level: dataLevel(props.coverage.tables_in[0]), depth: 2 }] : []),
        ] as Item[])
      : []),
    // Earned: present once the code declared a route.
    ...(props.coverage?.routes || props.coverage?.pages || props.coverage?.commands || props.coverage?.jobs
      ? ([{ label: 'Features', screen: 'features', depth: 1, badge: `${(props.coverage.routes ?? 0) + (props.coverage.pages ?? 0) + (props.coverage.commands ?? 0) + (props.coverage.jobs ?? 0)}` }] as Item[])
      : []),
    ...(props.coverage?.dependencies
      ? ([{ label: 'Dependencies', screen: 'deps', depth: 1, badge: String(props.coverage.dependencies) }] as Item[])
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
    // What the spec designs and the app does not do yet: a place in the tree, marked, so it is not forgotten.
    ...(published
      ? []
      : ([
          { label: 'Not built yet', depth: 0, header: true, divider: true },
          { label: 'Flows', screen: 'planned', depth: 1, planned: 'flows' },
          { label: 'Deployment', screen: 'planned', depth: 1, planned: 'deployment' },
          { label: 'Intent vs actual', screen: 'planned', depth: 1, planned: 'intent' },
          { label: 'Ask the map', screen: 'planned', depth: 1, planned: 'ask' },
          { label: 'Everything planned', screen: 'planned', depth: 1, badge: String(planned.length) },
        ] as Item[])),
  ]

  return (
    <nav className={`nav ${props.open ? 'open' : ''}`} aria-label="Sections">
      {items.map((it, i) => {
        const active = it.view
          ? inExplorer && level === it.view.level && (route.focus ?? '') === (it.view.focus ?? '') && (route.q ?? '') === (it.view.find ?? '') && (route.dim === '1') === !!it.view.dim
          : it.level
            ? inExplorer && (level === it.level || (it.label === 'Components' && (level.startsWith('component:') || level.startsWith('structure:'))) || (it.label === 'Data' && level.startsWith('data:')))
            : it.planned
              ? route.screen === 'planned' && route.focus === it.planned
              : it.screen === route.screen && !(it.screen === 'planned' && route.focus)
        return (
          <div key={i}>
            {it.divider && <div className="nav-divider" />}
            {it.header ? (
              <div className={`nav-item nav-header depth-${it.depth}`}>{it.label}</div>
            ) : (
              <button
                className={`nav-item depth-${it.depth} ${active ? 'active' : ''} ${it.planned ? 'nav-planned' : ''}`}
                title={it.planned ? 'Designed, not built yet' : undefined}
                onClick={() =>
                  go(
                    it.view
                      ? { screen: 'explorer', level: it.view.level, focus: it.view.focus, q: it.view.find, dim: it.view.dim ? '1' : undefined }
                      : it.level
                        ? { screen: 'explorer', level: it.level }
                        : { screen: it.screen!, focus: it.planned },
                  )
                }
              >
                <span className="nav-label">{it.label}</span>
                {it.badge && <span className="nav-badge">{it.badge}</span>}
                {it.planned && <span className="planned-tag">planned</span>}
              </button>
            )}
          </div>
        )
      })}
    </nav>
  )
}
