import { useEffect, useRef, useState } from 'react'
import type { CoverageResponse, Session, Summary, Version, ViewsResponse } from './api'
import { RootContext, published, useApi } from './api'
import { Failure, Loading } from './ui/marks'
import type { Screen } from './route'
import { useRoute } from './route'
import { useTheme } from './theme'
import { TopBar } from './shell/TopBar'
import { Nav } from './shell/Nav'
import { Palette } from './shell/Palette'
import { Shortcuts } from './shell/Shortcuts'
import { Explorer } from './screens/explorer/Explorer'
import { Overview } from './screens/Overview'
import { Coverage } from './screens/Coverage'
import { Features } from './screens/Features'
import { Changes } from './screens/Changes'
import { Corrections } from './screens/Corrections'
import { Documents } from './screens/Documents'
import { NetworkRuns } from './screens/NetworkRuns'

const titles: Record<Screen, string> = {
  overview: 'Overview',
  explorer: 'Explorer',
  features: 'Features',
  changes: 'Changes',
  docs: 'Documents',
  coverage: 'Coverage',
  rules: 'Corrections',
  runs: 'Network runs',
}

export function App() {
  const summary = useApi<Summary>('/api/summary')
  const versions = useApi<Version[]>('/api/versions')
  const [route, go] = useRoute()
  const [theme, nextTheme] = useTheme()
  const [navOpen, setNavOpen] = useState(false)
  const [viewsKey, setViewsKey] = useState(0)
  const [searching, setSearching] = useState(false)
  const coverage = useApi<CoverageResponse>('/api/coverage')
  const session = useApi<Session>('/api/session')
  const [keys, setKeys] = useState(false)
  const views = useApi<ViewsResponse>(`/api/views${viewsKey ? `#${viewsKey}` : ''}`)

  // The screen the explorer returns to.
  const prev = useRef<Screen>('overview')
  useEffect(() => {
    if (route.screen !== 'explorer') prev.current = route.screen
    setNavOpen(false)
  }, [route.screen])

  // ⌘K searches; E opens the explorer; Esc leaves it; ? shows the shortcuts. Never while typing.
  useEffect(() => {
    const on = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault()
        setSearching((s) => !s)
        return
      }
      const t = e.target as HTMLElement
      if (e.metaKey || e.ctrlKey || e.altKey || t.closest('input, select, textarea, [contenteditable]')) return
      if (searching || keys || document.querySelector('.dialog-scrim')) return
      if ((e.key === 'e' || e.key === 'E') && route.screen !== 'explorer') go({ screen: 'explorer' })
      else if (e.key === 'Escape' && route.screen === 'explorer') go({ screen: prev.current })
      else if (e.key === '/' && route.screen !== 'explorer') {
        e.preventDefault()
        setSearching(true)
      } else if (e.key === '?') setKeys(true)
    }
    window.addEventListener('keydown', on)
    return () => window.removeEventListener('keydown', on)
  }, [route.screen, go, searching, keys])

  useEffect(() => {
    const repo = summary.data?.name
    document.title = repo ? `${titles[route.screen]} · ${repo} · archdoc` : 'archdoc'
  }, [route.screen, summary.data?.name])

  if (summary.error) return <main className="boot"><Failure error={summary.error} /></main>
  if (!summary.data) return <main className="boot"><Loading /></main>
  const s = summary.data
  const version = route.v ?? null

  return (
    <RootContext.Provider value={{ root: s.root, remote: session.data?.remote, commit: session.data?.commit }}>
      <div className="shell">
        <TopBar
          summary={s}
          versions={versions.data ?? []}
          route={route}
          go={go}
          prev={titles[prev.current]}
          onBack={() => go({ screen: prev.current })}
          onMenu={() => setNavOpen((o) => !o)}
          theme={theme}
          onTheme={nextTheme}
          onSearch={() => setSearching(true)}
          session={session.data}
        />
        <div className="shell-body">
          <Nav summary={s} route={route} go={go} open={navOpen} views={views.data?.views ?? []} coverage={coverage.data} />
          {navOpen && <div className="nav-scrim" onClick={() => setNavOpen(false)} />}
          <main className={`shell-main ${route.screen === 'explorer' ? 'full' : 'page'}`}>
            {route.screen === 'overview' && <Overview summary={s} versions={versions.data ?? []} version={version} go={go} coverage={coverage.data} />}
            {route.screen === 'explorer' && (
              <Explorer version={version} versions={versions.data ?? []} route={route} go={go} editable={!published} onViewsChanged={() => setViewsKey((k) => k + 1)} />
            )}
            {route.screen === 'features' && <Features version={version} route={route} go={go} />}
            {route.screen === 'changes' && <Changes versions={versions.data ?? []} route={route} go={go} />}
            {route.screen === 'docs' && <Documents version={version} route={route} go={go} />}
            {route.screen === 'coverage' && <Coverage go={go} />}
            {route.screen === 'rules' && <Corrections />}
            {route.screen === 'runs' && <NetworkRuns route={route} go={go} />}
          </main>
        </div>
      </div>
      <Palette open={searching} onClose={() => setSearching(false)} go={go} views={views.data?.views ?? []} version={version} />
      {keys && <Shortcuts onClose={() => setKeys(false)} />}
    </RootContext.Provider>
  )
}
