import type { CoverageResponse, Summary } from '../api'
import { useCapabilities } from '../source'
import { planned } from '../planned'
import type { Route, Screen } from '../route'
import { sectionOf } from '../route'

interface Item {
  label: string
  /** The plain question the entry answers (platform-surface §4.3), as its title. */
  q?: string
  screen: Screen
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
export function Nav(props: { summary: Summary; route: Route; go: (r: Partial<Route>) => void; open: boolean; coverage?: CoverageResponse }) {
  const { route, go } = props
  const can = useCapabilities()
  const entries = (props.coverage?.routes ?? 0) + (props.coverage?.pages ?? 0) + (props.coverage?.commands ?? 0) + (props.coverage?.jobs ?? 0)
  const section = sectionOf(route.screen)

  const items: Item[] = [
    { label: 'Overview', q: 'What is this?', screen: 'overview' },
    { label: 'What changed', q: 'What just changed?', screen: 'changes' },
    ...(can.ask ? ([{ label: 'Did it build what I asked?', q: 'Did the code do what I asked?', screen: 'plan' }] as Item[]) : []),
    { label: 'Map', q: 'How is it put together?', screen: 'explorer', divider: true },
    ...(entries ? ([{ label: 'What it does', q: 'What can it do, and how does each thing work?', screen: 'features' }] as Item[]) : []),
    { label: 'Reference', q: 'Where is the complete detail?', screen: 'reference' },
    // Settings is the repository's own choices: locally, and later for a signed-in owner — a
    // published site or a visitor has none to change (§4.5).
    ...(can.place === 'local' ? ([{ label: 'Settings', q: 'How does this repository behave?', screen: 'rules', divider: true }] as Item[]) : []),
    // Development builds keep the list of what is designed and not built; the product never shows it (§4.4).
    ...(can.plans ? ([{ label: 'Not built yet', screen: 'planned', badge: String(planned.length), divider: true }] as Item[]) : []),
  ]

  return (
    <nav className={`nav ${props.open ? 'open' : ''}`} aria-label="Sections">
      {items.map((it, i) => {
        const active = it.screen === section
        return (
          <div key={i}>
            {it.divider && <div className="nav-divider" />}
            <button
              className={`nav-item ${active ? 'active' : ''} ${it.planned ? 'nav-planned' : ''}`}
              title={it.planned ? 'Designed, not built yet' : it.q}
              aria-current={active ? 'page' : undefined}
              onClick={() => go({ screen: it.screen, focus: it.planned })}
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
