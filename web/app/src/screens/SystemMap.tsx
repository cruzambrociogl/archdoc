import { useState } from 'react'
import type { SystemLink, SystemRepo, SystemResponse } from '../api'
import { RootContext } from '../api'
import { pinWords, systemSentence } from '../sentence'
import type { Crumb } from '../shell/parent'
import { WebsiteBar } from '../shell/WebsiteBar'
import { Cite } from '../ui/Cite'
import { Failure, Loading } from '../ui/marks'

const W = 230
const H = 88
const COL = 130
const ROW = 28

const day = (iso: string) => new Date(iso).toLocaleDateString(undefined, { day: 'numeric', month: 'short' })
/** A package's short name on an arrow: a Go module's last segment, an npm package whole. */
const short = (pkg: string) => (pkg.includes('/') && !pkg.startsWith('@') ? pkg.slice(pkg.lastIndexOf('/') + 1) : pkg)

/**
 * The System map (platform-surface §4.6): how do my repositories fit together? One sentence written
 * from the links; the repositories as boxes, the links as arrows from the one that requires to the
 * one required, drift on the arrow; and every link with the lines at both ends — which is also the
 * picture's text alternative (§4.8). A box opens that repository.
 */
export function SystemMap(props: { sys?: SystemResponse; error?: Error; missing?: string; above?: Crumb[]; onOpen: (repo: string) => void; onSystem: () => void }) {
  const { sys } = props
  const [chosen, setChosen] = useState<number | null>(null)

  return (
    <div className="shell">
      <WebsiteBar onStart={props.onSystem}>
        {sys && (
          <nav className="crumbs" aria-label="Where you are">
            {(props.above ?? []).map((c) => (
              <span key={c.label} className="crumb-step">
                <button className="crumb mono" onClick={c.open}>
                  {c.label}
                </button>
                <span className="crumb-sep">›</span>
              </span>
            ))}
            <span className="crumb current mono" aria-current="page">
              {sys.name}
            </span>
          </nav>
        )}
      </WebsiteBar>
      <div className="shell-body">
        <nav className="nav" aria-label="Sections">
          <button className="nav-item active" aria-current="page" title="How do my repositories fit together?">
            <span className="nav-label">System map</span>
          </button>
        </nav>
        <div className="main-col">
          <main className="shell-main page">
            <div className="reading wide system-map">
              {props.error ? (
                <Failure error={props.error.message} whole />
              ) : !sys ? (
                <Loading q="How do my repositories fit together?" what="the repositories" />
              ) : (
                <>
                  <div className="page-q">How do my repositories fit together?</div>
                  {props.missing && <p className="gone-more">There’s no repository {props.missing} in {sys.name}. These are the ones it holds:</p>}
                  <p className="overview-answer">{systemSentence(sys).join(' ')}</p>
                  <Picture sys={sys} chosen={chosen} onChoose={setChosen} onOpen={props.onOpen} />
                  <Evidence sys={sys} chosen={chosen} onChoose={setChosen} />
                </>
              )}
            </div>
          </main>
        </div>
      </div>
    </div>
  )
}

/** Repositories in columns: those that require others to the left of what they require. */
function layout(sys: SystemResponse) {
  const out = new Map<string, Set<string>>()
  for (const l of sys.links) out.set(l.from, (out.get(l.from) ?? new Set()).add(l.to))
  const depth = (name: string, seen: Set<string>): number => {
    if (seen.has(name)) return 0 // a cycle: drawn, not followed
    const next = [...(out.get(name) ?? [])]
    return next.length ? 1 + Math.max(...next.map((n) => depth(n, new Set(seen).add(name)))) : 0
  }
  const layers = new Map(sys.repos.map((r) => [r.name, depth(r.name, new Set())]))
  const deepest = Math.max(0, ...layers.values())
  const columns = new Map<number, SystemRepo[]>()
  for (const r of sys.repos) {
    const c = deepest - layers.get(r.name)!
    columns.set(c, [...(columns.get(c) ?? []), r])
  }
  const tallest = Math.max(...[...columns.values()].map((c) => c.length))
  const at = new Map<string, { x: number; y: number }>()
  for (const [c, repos] of columns) {
    const top = ((tallest - repos.length) * (H + ROW)) / 2 // a shorter column centred on the tallest
    repos.forEach((r, i) => at.set(r.name, { x: c * (W + COL), y: top + i * (H + ROW) }))
  }
  return { at, width: (deepest + 1) * W + deepest * COL, height: tallest * H + (tallest - 1) * ROW }
}

function Picture(props: { sys: SystemResponse; chosen: number | null; onChoose: (i: number | null) => void; onOpen: (repo: string) => void }) {
  const { sys } = props
  const { at, width, height } = layout(sys)
  // Links between the same two repositories draw side by side; five and more, as one arrow with a count (§4.6).
  const pairs = new Map<string, number[]>()
  sys.links.forEach((l, i) => pairs.set(`${l.from}\u0000${l.to}`, [...(pairs.get(`${l.from}\u0000${l.to}`) ?? []), i]))
  const arrows: { d: string; label: string; mid: { x: number; y: number }; links: number[] }[] = []
  for (const idx of pairs.values()) {
    const l = sys.links[idx[0]]
    const a = at.get(l.from)!
    const b = at.get(l.to)!
    const groups = idx.length >= 5 ? [idx] : idx.map((i) => [i])
    groups.forEach((g, k) => {
      const dy = (k - (groups.length - 1) / 2) * 26
      const x1 = a.x + W
      const y1 = a.y + H / 2 + dy
      const x2 = b.x
      const y2 = b.y + H / 2 + dy
      const mx = (x1 + x2) / 2
      const one = sys.links[g[0]]
      const behind = one.pin && one.pin.ahead > 0 ? ` · ${one.pin.ahead} behind` : ''
      arrows.push({ d: `M${x1} ${y1} C${mx} ${y1} ${mx} ${y2} ${x2 - 2} ${y2}`, label: g.length > 1 ? `${g.length} packages` : short(one.package) + behind, mid: { x: mx, y: (y1 + y2) / 2 }, links: g })
    })
  }

  return (
    <div className="system-picture">
      <div className="system-canvas" style={{ width, height: height + 8 }}>
        <svg className="system-arrows" width={width} height={height + 8} aria-hidden="true">
          <defs>
            <marker id="system-head" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse">
              <path d="M0 0 L10 5 L0 10 z" />
            </marker>
          </defs>
          {arrows.map((ar, i) => {
            const on = props.chosen !== null && ar.links.includes(props.chosen)
            return (
              <g key={i} className={`system-arrow ${on ? 'on' : ''}`} onClick={() => props.onChoose(on ? null : ar.links[0])}>
                <path d={ar.d} markerEnd="url(#system-head)" />
                <path d={ar.d} className="system-arrow-hit" />
                <text x={ar.mid.x} y={ar.mid.y - 6} textAnchor="middle">
                  {ar.label}
                </text>
              </g>
            )
          })}
        </svg>
        {sys.repos.map((r) => {
          const p = at.get(r.name)!
          return (
            <button key={r.name} className="system-box" style={{ left: p.x, top: p.y, width: W, height: H }} onClick={() => props.onOpen(r.name)} aria-label={`${r.name} — open its overview`}>
              <span className="system-box-name mono">{r.name}</span>
              <span className="system-box-what">{r.description ?? `${r.applications} ${r.applications === 1 ? 'application' : 'applications'}`}</span>
              <span className="system-box-status mono">
                version {r.version} · {r.commit.slice(0, 7)} · {day(r.created_at)}
              </span>
            </button>
          )
        })}
      </div>
    </div>
  )
}

/** Every link, with the line that requires and the line that declares — the evidence, and the picture as a list. */
function Evidence(props: { sys: SystemResponse; chosen: number | null; onChoose: (i: number | null) => void }) {
  const root = new Map(props.sys.repos.map((r) => [r.name, r.root]))
  if (props.sys.links.length === 0) return null
  return (
    <section className="system-links" aria-label="Every link, with the lines at both ends">
      <div className="page-q">Every link</div>
      <ul>
        {props.sys.links.map((l: SystemLink, i) => (
          <li key={i} className={props.chosen === i ? 'on' : ''} onClick={() => props.onChoose(props.chosen === i ? null : i)}>
            <div>
              <span className="mono">{l.from}</span> requires <span className="mono">{l.to}</span>’s <span className="mono">{l.package}</span>
              {pinWords(l)}.
            </div>
            <div className="system-link-cites small">
              <RootContext.Provider value={{ root: root.get(l.from) ?? '' }}>
                <span className="muted">required at</span> <Cite p={l.uses} />
              </RootContext.Provider>
              <RootContext.Provider value={{ root: root.get(l.to) ?? '' }}>
                <span className="muted">named at</span> <Cite p={l.declares} />
              </RootContext.Provider>
            </div>
          </li>
        ))}
      </ul>
    </section>
  )
}
