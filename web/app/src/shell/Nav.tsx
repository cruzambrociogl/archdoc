import type { CoverageResponse, SavedView, Summary } from '../api'
import { useCapabilities } from '../source'
import { planned } from '../planned'
import { componentLevel, dataLevel } from '../api'
import type { Route, Screen } from '../route'
import { sectionOf } from '../route'

interface Item {
  label: string
  /** The plain question the entry answers (platform-surface §4.3), as its title. */
  q?: string
  screen?: Screen
  level?: string
  view?: SavedView
  depth: 0 | 1
  badge?: string
  divider?: boolean
  /** Designed, not built: drawn dashed, and opens the list of what is still to build. */
  planned?: string
}

/**
 * The left menu, at the repository level (platform-surface §4.3): six entries and Settings, named
 * in plain words, each answering one question. An entry appears only where the place can do it
 * (§4.5) and when there is evidence behind it (surface-spec §2, rule 3) — absent, never disabled.
 */
export function Nav(props: { summary: Summary; route: Route; go: (r: Partial<Route>) => void; open: boolean; views: SavedView[]; coverage?: CoverageResponse }) {
  const { summary: s, route, go } = props
  const can = useCapabilities()
  const inMap = route.screen === 'explorer'
  const level = route.level ?? 'context'
  const firstInside = props.coverage?.code?.find((c) => c.read && c.container && c.components > 0)?.container
  const entries = (props.coverage?.routes ?? 0) + (props.coverage?.pages ?? 0) + (props.coverage?.commands ?? 0) + (props.coverage?.jobs ?? 0)
  const section = sectionOf(route.screen)

  const items: Item[] = [
    { label: 'Overview', q: 'What is this?', screen: 'overview', depth: 0 },
    { label: 'What changed', q: 'What just changed?', screen: 'changes', depth: 0 },
    ...(can.plans ? ([{ label: 'Did it build what I asked?', q: 'Did the code do what I asked?', screen: 'plan', depth: 0, planned: 'intent' }] as Item[]) : []),
    { label: 'Map', q: 'How is it put together?', screen: 'explorer', depth: 0, divider: true },
    // Until the Map carries its levels and saved views on the canvas (stage 1, part 3), they sit under it.
    ...(inMap
      ? ([
          { label: 'The outside world', screen: 'explorer', level: 'context', depth: 1 },
          ...(s.tiny ? [] : [{ label: 'Apps and services', screen: 'explorer', level: 'container', depth: 1 } as Item]),
          ...(firstInside ? [{ label: 'Inside one part', screen: 'explorer', level: componentLevel(firstInside), depth: 1 } as Item] : []),
          ...(props.coverage?.tables_in?.length ? [{ label: 'What it stores', screen: 'explorer', level: dataLevel(props.coverage.tables_in[0]), depth: 1 } as Item] : []),
          ...(s.dataflow ? [{ label: 'How data moves', screen: 'explorer', level: 'dataflow', depth: 1 } as Item] : []),
          ...props.views.map((v) => ({ label: v.name, view: v, depth: 1, badge: 'view' }) as Item),
        ] as Item[])
      : []),
    ...(entries ? ([{ label: 'What it does', q: 'What can it do, and how does each thing work?', screen: 'features', depth: 0 }] as Item[]) : []),
    { label: 'Reference', q: 'Where is the complete detail?', screen: 'reference', depth: 0 },
    // Settings is the repository's own choices; a published site has none to change (§4.5).
    ...(can.place !== 'published' ? ([{ label: 'Settings', q: 'How does this repository behave?', screen: 'rules', depth: 0, divider: true }] as Item[]) : []),
    // Development builds keep the list of what is designed and not built; the product never shows it (§4.4).
    ...(can.plans ? ([{ label: 'Not built yet', screen: 'planned', depth: 0, badge: String(planned.length), divider: true }] as Item[]) : []),
  ]

  return (
    <nav className={`nav ${props.open ? 'open' : ''}`} aria-label="Sections">
      {items.map((it, i) => {
        const active = it.view
          ? inMap && level === it.view.level && (route.focus ?? '') === (it.view.focus ?? '') && (route.q ?? '') === (it.view.find ?? '') && (route.dim === '1') === !!it.view.dim
          : it.level
            ? inMap && (level === it.level || (it.level.startsWith('component:') && (level.startsWith('component:') || level.startsWith('structure:') || level.startsWith('main:'))) || (it.level.startsWith('data:') && level.startsWith('data:')))
            : it.depth === 0 && it.screen === section
        return (
          <div key={i}>
            {it.divider && <div className="nav-divider" />}
            <button
              className={`nav-item depth-${it.depth} ${active ? 'active' : ''} ${it.planned ? 'nav-planned' : ''}`}
              title={it.planned ? 'Designed, not built yet' : it.q}
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
          </div>
        )
      })}
    </nav>
  )
}
