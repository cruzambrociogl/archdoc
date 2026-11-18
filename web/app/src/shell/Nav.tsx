import { useState } from 'react'
import type { CoverageResponse, Summary } from '../api'
import { useCapabilities } from '../source'
import type { Route, Screen } from '../route'
import { sectionOf, titles } from '../route'

interface Entry {
  screen: Screen
  /** The plain question the entry answers (platform-surface §17.3), as its hover title. */
  q: string
  /** A count on the entry: on a return visit, the versions since this reader last looked. */
  badge?: number
}

interface Group {
  /** The group's name; Home has none. */
  label?: string
  entries: Entry[]
}

// One icon per section, 16 by 16, drawn in a single stroke (the design, `Nav F - Sections`).
const icons: Partial<Record<Screen, string>> = {
  home: 'M2.5 7.5L8 2.5l5.5 5v6H10v-4H6v4H2.5z',
  architecture: 'M1.5 2.5h5v4h-5zM9.5 9.5h5v4h-5zM4 6.5v5h5.5',
  data: 'M3 4.2c0-1 2.2-1.7 5-1.7s5 .7 5 1.7v7.6c0 1-2.2 1.7-5 1.7s-5-.7-5-1.7zM3 4.2c0 1 2.2 1.7 5 1.7s5-.7 5-1.7M3 8c0 1 2.2 1.7 5 1.7s5-.7 5-1.7',
  features: 'M2.5 2.5H7V7H2.5zM9 2.5h4.5V7H9zM2.5 9H7v4.5H2.5zM9 9h4.5v4.5H9z',
  usage: 'M2 3h12v8H2zM6 13.5h4M8 11v2.5',
  changes: 'M8 2v4M8 10v4M5 8a3 3 0 106 0 3 3 0 00-6 0',
  plan: 'M2.5 8.5l3.5 3.5 7.5-8',
  reference: 'M3 2.5h8.5v11H3zM5.5 5.5h4M5.5 8h4M5.5 10.5h2.5',
  settings: 'M8 5.5a2.5 2.5 0 100 5 2.5 2.5 0 000-5zM8 1.5v2M8 12.5v2M1.5 8h2M12.5 8h2M3.4 3.4l1.4 1.4M11.2 11.2l1.4 1.4M3.4 12.6l1.4-1.4M11.2 4.8l1.4-1.4',
}

const NARROW = 'archdoc.menu.narrow'
const readNarrow = () => {
  try {
    return localStorage.getItem(NARROW) === '1'
  } catch {
    return false
  }
}

/**
 * The side menu (platform-surface §17.3): single words with an icon each, in four groups — at most
 * three entries a group, four groups (rule 15). Narrowed, it keeps the icons and the groups become
 * separators. An entry appears only where the place can do it and there is evidence behind it —
 * absent, never disabled.
 */
export function Nav(props: { summary: Summary; route: Route; go: (r: Partial<Route>) => void; open: boolean; coverage?: CoverageResponse; newer?: number }) {
  const { route, go } = props
  const can = useCapabilities()
  const cov = props.coverage
  const entries = (cov?.routes ?? 0) + (cov?.pages ?? 0) + (cov?.commands ?? 0) + (cov?.jobs ?? 0)
  const section = sectionOf(route.screen)
  const [narrow, setNarrow] = useState(readNarrow)
  const toggle = () => {
    setNarrow(!narrow)
    try {
      localStorage.setItem(NARROW, narrow ? '0' : '1')
    } catch {
      // A private window keeps no setting; the menu still narrows for this visit.
    }
  }

  const groups: Group[] = [
    { entries: [{ screen: 'home', q: 'What is this, and what is new?' }] },
    {
      label: 'How it’s built',
      entries: [
        { screen: 'architecture', q: 'How is it put together? The C4 diagrams' },
        ...(cov?.tables ? [{ screen: 'data' as const, q: 'What does it store, and how does data move?' }] : []),
      ],
    },
    {
      label: 'What it does',
      entries: entries
        ? [
            { screen: 'features', q: 'What is it for?' },
            { screen: 'usage', q: 'How is it used?' },
          ]
        : [],
    },
    {
      label: 'Keeping up',
      entries: [{ screen: 'changes', q: 'What changed since you last looked?', badge: props.newer || undefined }, ...(can.ask ? [{ screen: 'plan' as const, q: 'Did it build what you asked?' }] : [])],
    },
    { label: 'In detail', entries: [{ screen: 'reference', q: 'Every part, route and table' }] },
  ]

  const item = (e: Entry) => {
    const on = e.screen === section
    return (
      <button
        key={e.screen}
        className={`nav-item ${on ? 'active' : ''}`}
        title={e.q}
        aria-label={titles[e.screen]}
        aria-current={on ? 'page' : undefined}
        onClick={() => go({ screen: e.screen })}
      >
        <svg className="nav-icon" aria-hidden="true" width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.4">
          <path d={icons[e.screen]} />
        </svg>
        <span className="nav-label">{titles[e.screen]}</span>
        {e.badge ? (
          <span className="nav-badge" title={`${e.badge} ${e.badge === 1 ? 'version' : 'versions'} since you last looked`}>
            {e.badge}
          </span>
        ) : null}
      </button>
    )
  }

  return (
    <nav className={`nav ${props.open ? 'open' : ''} ${narrow ? 'narrow' : ''}`} aria-label="Sections">
      {groups
        .filter((g) => g.entries.length > 0)
        .map((g, i) => (
          <div key={i} className="nav-group" role="group" aria-label={g.label}>
            {g.label && <div className="nav-header">{g.label}</div>}
            {g.label && <div className="nav-sep" />}
            {g.entries.map(item)}
          </div>
        ))}
      <div className="nav-fill" />
      {/* Settings is the repository's own choices: locally, and later for a signed-in owner — a
          published site or a visitor has none to change. */}
      {can.place === 'local' && item({ screen: 'settings', q: 'How does this repository behave?' })}
      <button className="nav-item nav-toggle" onClick={toggle} aria-label={narrow ? 'Widen the menu' : 'Narrow the menu'} title={narrow ? 'Widen the menu' : 'Narrow the menu'}>
        <span className="nav-icon mono" aria-hidden="true">
          {narrow ? '»' : '«'}
        </span>
        <span className="nav-label">Narrow menu</span>
      </button>
    </nav>
  )
}
