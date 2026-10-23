import { useEffect, useRef, useState } from 'react'
import type { Session, Summary, Version } from '../api'
import { useCapabilities, useSource } from '../source'
import type { Route } from '../route'
import { levelName, sectionOf, titles } from '../route'
import type { Theme } from '../theme'
import { Planned } from '../ui/Planned'
import { TruthMark } from '../ui/marks'
import type { MarkKind } from '../ui/markInfo'
import { MarkGlyph } from '../ui/markInfo'
import { useParent } from './parent'

export const themeLabel: Record<Theme, string> = { system: 'Theme: system', light: 'Theme: light', dark: 'Theme: dark' }
export const themeGlyph: Record<Theme, string> = { system: '◐', light: '○', dark: '●' }

type Menu = 'status' | 'share' | 'more' | null

/** How long the read that made a version took, on the website: "9 s", "2 min 5 s". */
function took(v?: Version): string {
  if (!v?.started_at) return ''
  const s = Math.round((Date.parse(v.created_at) - Date.parse(v.started_at)) / 1000)
  if (!(s >= 0)) return ''
  return s < 60 ? `${s} s` : `${Math.floor(s / 60)} min ${s % 60} s`
}

const day = (iso?: string) => (iso ? new Date(iso).toLocaleDateString(undefined, { day: 'numeric', month: 'short' }) : '')

/**
 * The top bar (platform-surface §4.3): where you are, whether it is current, one box to search, and
 * what belongs to the page. Each part appears only where the place can do it (§4.5). What is designed
 * and not built — updating from the app, publishing, this machine's settings — shows only in a
 * development build, as a dashed placeholder.
 */
export function TopBar(props: {
  summary: Summary
  versions: Version[]
  route: Route
  go: (r: Partial<Route>, o?: { keep?: boolean; replace?: boolean }) => void
  onMenu: () => void
  theme: Theme
  onTheme: () => void
  onSearch: () => void
  onKeys: () => void
  session?: Session
  /** A part's name by its id, for "Inside immich-server". */
  name: (id: string) => string
  /** Opens the rail on a mark: what it means, and what you can do about it. */
  onMark: (mark: MarkKind) => void
}) {
  const can = useCapabilities()
  const parent = useParent()
  const { summary: s, route, go } = props
  const [open, setOpen] = useState<Menu>(null)
  const bar = useRef<HTMLElement>(null)
  const toggle = (m: Menu) => setOpen((o) => (o === m ? null : m))

  // One menu at a time; a click outside or Esc closes it.
  useEffect(() => {
    if (!open) return
    const out = (e: MouseEvent) => bar.current && !bar.current.contains(e.target as Node) && setOpen(null)
    const esc = (e: KeyboardEvent) => e.key === 'Escape' && setOpen(null)
    document.addEventListener('mousedown', out)
    document.addEventListener('keydown', esc)
    return () => {
      document.removeEventListener('mousedown', out)
      document.removeEventListener('keydown', esc)
    }
  }, [open])
  useEffect(() => setOpen(null), [route.screen])

  // Which version is on screen, and when it was made.
  const latest = props.versions.find((v) => v.id === s.latest_version)
  const shown = route.v ? props.versions.find((v) => v.id === route.v) : latest
  const older = !!route.v && route.v !== s.latest_version
  const commit = (older ? shown?.commit : (props.session?.commit ?? s.commit)) ?? ''
  const made = older ? shown?.created_at : (props.session?.generated_at ?? latest?.created_at)

  // Where you are: the repository, the section, and the page or level inside it.
  const section = sectionOf(route.screen)
  const crumbs: { label: string; mono?: boolean; on?: () => void }[] = [
    ...(parent?.crumbs ?? []).map((c) => ({ label: c.label, mono: true, on: c.open })),
    { label: parent?.repo ?? s.name, mono: true, on: () => go({ screen: 'overview' }) },
    { label: titles[section], on: route.screen !== section || (section === 'explorer' && route.level) ? () => go({ screen: section }) : undefined },
    ...(route.screen !== section ? [{ label: titles[route.screen] }] : []),
    ...(route.screen === 'explorer' && route.level ? [{ label: levelName(route.level, props.name) }] : []),
  ]

  return (
    <header className="topbar" ref={bar}>
      <button className="topbar-menu" onClick={props.onMenu} title="Menu" aria-label="Menu">
        ☰
      </button>
      <button className="brand-mark" onClick={() => go({ screen: 'overview' })}>
        archdoc
      </button>

      <nav className="crumbs" aria-label="Where you are">
        {crumbs.map((c, i) => (
          <span key={i} className="crumb-step">
            {i > 0 && <span className="crumb-sep">›</span>}
            {c.on ? (
              <button className={`crumb ${c.mono ? 'mono' : ''}`} onClick={c.on}>
                {c.label}
              </button>
            ) : (
              <span className={`crumb current ${c.mono ? 'mono' : ''}`} aria-current="page">
                {c.label}
              </span>
            )}
          </span>
        ))}
      </nav>

      {/* Is this up to date? Which commit, and when; an older version says so. */}
      <div className="pop-anchor status-anchor">
        <button className={`status-pill ${older ? 'older' : ''}`} onClick={() => toggle('status')} aria-expanded={open === 'status'}>
          <span className="status-glyph" aria-hidden="true">
            {older ? '↺' : ''}
          </span>
          <span className="status-text">
            {older ? `Showing v${route.v}` : 'From'} <span className="mono">{commit.slice(0, 7) || 'no commit'}</span>
            {made && !older ? ` · ${day(made)}` : ''}
          </span>
          <span className="caret">▾</span>
        </button>
        {open === 'status' && (
          <div className="pop status-pop">
            <div className="pop-q">Is this up to date?</div>
            <p className="pop-sentence">
              {older ? (
                <>
                  You are looking at version {route.v}, from commit <span className="mono">{commit.slice(0, 7)}</span> on {day(made)}. The latest is version {s.latest_version}.
                </>
              ) : can.place === 'published' ? (
                <>
                  This site was built from commit <span className="mono">{commit.slice(0, 7)}</span>
                  {made ? <> on {day(made)}</> : null}. It shows that version, and what changed since the one before it.
                </>
              ) : can.place === 'hosted' ? (
                <>
                  These docs were generated from commit <span className="mono">{commit.slice(0, 7)}</span>
                  {made ? <> on {day(made)}</> : null}, when someone asked for them{took(shown) ? `; reading it took ${took(shown)}` : ''}. A later
                  commit is documented the next time someone asks.
                </>
              ) : (
                <>
                  These docs were generated from commit <span className="mono">{commit.slice(0, 7) || '—'}</span>
                  {made ? <> on {day(made)}</> : null}. Whether the code changed since, the app can't tell yet: <span className="mono">archdoc status</span> in a terminal says.
                </>
              )}
            </p>
            <div className="pop-actions">
              {older && (
                <button className="btn btn-ink" onClick={() => go({ v: undefined }, { keep: true })}>
                  Show the latest
                </button>
              )}
              {can.history && props.versions.length > 1 && (
                <button className="link-btn" onClick={() => go({ screen: 'changes', v: route.v })}>
                  Every version →
                </button>
              )}
              <Planned id="freshness" go={go}>
                Up to date?
              </Planned>
              {can.update ? (
                <UpdateButton />
              ) : (
                <Planned id="run" go={go}>
                  Update
                </Planned>
              )}
              <Planned id="git" go={go}>
                Uncommitted
              </Planned>
            </div>
          </div>
        )}
      </div>

      <div className="topbar-spacer">
        {/* Where a model can be asked, the box says so — Search and Ask are one box (§4.3), and a box that
            only says "search" hides the other half. ◇ is the mark for what a model writes, here as everywhere. */}
        <button className="search-box" onClick={props.onSearch} title={can.ask ? 'Search, or ask a question (⌘K)' : 'Search (⌘K)'}>
          <svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" aria-hidden="true">
            <path d="M7 12A5 5 0 1 0 7 2a5 5 0 0 0 0 10z M10.5 10.5L14 14" />
          </svg>
          <span className="search-text">{can.ask ? 'Search, or ask a question…' : 'Search parts, routes, tables and pages…'}</span>
          {can.ask && (
            <span className="search-ai" title="AI can answer a question here, from what archdoc read — the cost is shown before anything is sent">
              <TruthMark state="interpreted" /> AI
            </span>
          )}
          <kbd>⌘K</kbd>
        </button>
      </div>

      {/* Share, where showing someone works today: a published or hosted page's address. Publishing from here is designed, not built. */}
      {can.place !== 'local' ? (
        <div className="pop-anchor">
          <button className={`btn btn-outline ${open === 'share' ? 'on' : ''}`} onClick={() => toggle('share')} aria-expanded={open === 'share'}>
            Share
          </button>
          {open === 'share' && <SharePop />}
        </div>
      ) : (
        <Planned id="publish" go={go}>
          Share
        </Planned>
      )}

      <button className="icon-btn" onClick={props.onTheme} title={themeLabel[props.theme]} aria-label={themeLabel[props.theme]}>
        {themeGlyph[props.theme]}
      </button>

      <div className="pop-anchor">
        <button className={`icon-btn more ${open === 'more' ? 'on' : ''}`} onClick={() => toggle('more')} aria-label="More" aria-expanded={open === 'more'}>
          ⋯
        </button>
        {open === 'more' && (
          <div className="pop more-pop">
            <button className="more-item" onClick={() => go(can.history ? { screen: 'changes', v: route.v } : { screen: 'about' })}>
              <span>
                <span className="more-label">Showing · version {shown?.id ?? s.latest_version}</span>
                <span className="more-sub">{can.history ? 'Every version, in What changed' : 'The one this site was built from'}</span>
              </span>
              <span className="mono more-right">{commit.slice(0, 7)}</span>
            </button>
            <button className="more-item" onClick={() => go({ screen: 'about' })}>
              <span>
                <span className="more-label">About these docs</span>
                <span className="more-sub">What archdoc could and couldn't read</span>
              </span>
            </button>
            <button className="more-item" onClick={() => (setOpen(null), props.onKeys())}>
              <span className="more-label">Keyboard shortcuts</span>
              <kbd>?</kbd>
            </button>
            <div className="more-legend">
              <div className="pop-q">What the marks mean</div>
              {(
                [
                  ['proven', 'Proven — read from the code, with the line that shows it'],
                  ['ai', 'Written by AI — from those facts, and marked'],
                  ['unres', 'Couldn’t be resolved — seen, not followed'],
                  ['notread', 'Not read — code archdoc doesn’t read yet'],
                ] as [MarkKind, string][]
              ).map(([k, text]) => (
                <button key={k} className="legend-row" onClick={() => (setOpen(null), props.onMark(k))}>
                  <MarkGlyph kind={k} /> {text}
                </button>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* The "you" slot (§4.3): locally, this machine's settings — designed, not built. A published site has none. */}
      {can.place === 'local' && (
        <Planned id="settings" go={go}>
          This machine
        </Planned>
      )}
    </header>
  )
}

/** How do I show someone? On a published site: this page's address, which opens it at this version. */
function SharePop() {
  const [copied, setCopied] = useState(false)
  const href = location.href
  return (
    <div className="pop share-pop">
      <div className="pop-q">How do I show someone?</div>
      <p className="pop-sentence">Send this address. It opens this page, as it is here.</p>
      <div className="copy-row">
        <span className="mono copy-text">{href}</span>
        <button
          className="btn btn-ink"
          onClick={() =>
            navigator.clipboard?.writeText(href).then(
              () => setCopied(true),
              () => setCopied(false),
            )
          }
        >
          {copied ? 'Copied' : 'Copy'}
        </button>
      </div>
    </div>
  )
}

/**
 * Update (§4.3): regenerate this repository's documentation, free and offline — where the place can
 * (capabilities.update, false everywhere today). The run is followed in the notice band.
 */
function UpdateButton() {
  const source = useSource()
  const [state, setState] = useState<{ asking?: boolean; error?: string; started?: boolean }>({})
  return (
    <>
      <button
        className="btn btn-ink"
        disabled={state.asking || state.started}
        onClick={async () => {
          setState({ asking: true })
          try {
            await source.act('/api/update')
            setState({ started: true })
          } catch (e) {
            setState({ error: e instanceof Error ? e.message : String(e) })
          }
        }}
      >
        {state.started ? 'Updating…' : 'Update'}
      </button>
      {state.error && <span className="pop-sentence">{state.error}</span>}
    </>
  )
}

