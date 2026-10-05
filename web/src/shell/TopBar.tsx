import type { Session, Summary, Version } from '../api'
import { published } from '../api'
import type { Route } from '../route'
import type { Theme } from '../theme'

const themeLabel: Record<Theme, string> = { system: 'Theme: system', light: 'Theme: light', dark: 'Theme: dark' }
const themeGlyph: Record<Theme, string> = { system: '◐', light: '○', dark: '●' }

/**
 * The top bar. It carries only controls that work today: the status pill and the run, publish,
 * git and settings actions arrive with the features behind them (surface-spec §11), not before as
 * dead buttons.
 */
export function TopBar(props: {
  summary: Summary
  versions: Version[]
  route: Route
  go: (r: Partial<Route>, o?: { keep?: boolean; replace?: boolean }) => void
  prev: string
  onBack: () => void
  onMenu: () => void
  theme: Theme
  onTheme: () => void
  onSearch: () => void
  session?: Session
}) {
  const { summary: s, route, go } = props
  const inExplorer = route.screen === 'explorer'

  return (
    <header className="topbar">
      <button className="topbar-menu" onClick={props.onMenu} title="Navigation" aria-label="Navigation">
        ☰
      </button>
      <div className="brand">
        <span className="brand-mark">archdoc</span>
        <span className="brand-repo">/ {s.name}</span>
      </div>

      {published ? (
        <span className="generated-from" title="This published site shows one version, built from the committed record">
          Generated from <span className="mono">{(props.session?.commit ?? s.commit ?? '').slice(0, 7) || 'an uncommitted tree'}</span>
          {props.session?.generated_at && <> on {new Date(props.session.generated_at).toLocaleDateString(undefined, { day: 'numeric', month: 'short', year: 'numeric' })}</>}
        </span>
      ) : (
      <label className="version-pick" title="The version on screen">
        <span className="mono">⌁ {s.commit ? s.commit.slice(0, 7) : 'no commit'}</span>
        <select
          value={route.v ?? ''}
          onChange={(e) => go({ v: e.target.value ? Number(e.target.value) : undefined }, { keep: true })}
        >
          <option value="">v{s.latest_version} · latest</option>
          {[...props.versions]
            .sort((a, b) => b.id - a.id)
            .filter((v) => v.id !== s.latest_version)
            .map((v) => (
              <option key={v.id} value={v.id}>
                v{v.id} · {new Date(v.created_at).toLocaleDateString()}
                {v.commit ? ` · ${v.commit.slice(0, 7)}` : ''}
              </option>
            ))}
        </select>
      </label>
      )}

      <div className="topbar-spacer">
        <button className="search-box" onClick={props.onSearch} title="Search everything (⌘K)">
          <svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6">
            <path d="M7 12A5 5 0 1 0 7 2a5 5 0 0 0 0 10z M10.5 10.5L14 14" />
          </svg>
          <span className="search-text">Search elements, files, documents…</span>
          <kbd>⌘K</kbd>
        </button>
      </div>

      {inExplorer ? (
        <button className="btn btn-outline" onClick={props.onBack} title="Leave the explorer (Esc)">
          ← <span className="btn-text">Back to {props.prev}</span> <kbd>Esc</kbd>
        </button>
      ) : (
        <button className="btn btn-ink" onClick={() => go({ screen: 'explorer' })} title="Open the architecture explorer (E)">
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
            <path d="M2 3h5v4H2z M9 9h5v4H9z M4.5 7v4H9" />
          </svg>
          <span className="btn-text">Explorer</span> <kbd>E</kbd>
        </button>
      )}

      <button className="icon-btn" onClick={props.onTheme} title={themeLabel[props.theme]} aria-label={themeLabel[props.theme]}>
        {themeGlyph[props.theme]}
      </button>
      {published ? (
        <span className="mode-badge published" title="A static site built by archdoc export --site: one version, no controls">
          Published
        </span>
      ) : (
        <span className="mode-badge" title="Served by archdoc serve on this machine">
          Local
        </span>
      )}
    </header>
  )
}
