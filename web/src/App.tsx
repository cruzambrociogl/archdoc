import { useEffect, useRef, useState } from 'react'
import type { Summary, Version } from './api'
import { RootContext, useApi } from './api'
import { Failure, Loading } from './ui/marks'
import type { Screen } from './route'
import { useRoute } from './route'
import { useTheme } from './theme'
import { TopBar } from './shell/TopBar'
import { Nav } from './shell/Nav'
import { Explorer } from './screens/explorer/Explorer'
import { Overview } from './screens/Overview'
import { Changes } from './screens/Changes'
import { Corrections } from './screens/Corrections'
import { Documents } from './screens/Documents'
import { NetworkRuns } from './screens/NetworkRuns'

const titles: Record<Screen, string> = {
  overview: 'Overview',
  explorer: 'Explorer',
  changes: 'Changes',
  docs: 'Documents',
  rules: 'Corrections',
  runs: 'Network runs',
}

export function App() {
  const summary = useApi<Summary>('/api/summary')
  const versions = useApi<Version[]>('/api/versions')
  const [route, go] = useRoute()
  const [theme, nextTheme] = useTheme()
  const [navOpen, setNavOpen] = useState(false)

  // The screen the explorer returns to.
  const prev = useRef<Screen>('overview')
  useEffect(() => {
    if (route.screen !== 'explorer') prev.current = route.screen
    setNavOpen(false)
  }, [route.screen])

  // E opens the explorer; Esc leaves it. Never while typing.
  useEffect(() => {
    const on = (e: KeyboardEvent) => {
      const t = e.target as HTMLElement
      if (e.metaKey || e.ctrlKey || e.altKey || t.closest('input, select, textarea, [contenteditable]')) return
      if ((e.key === 'e' || e.key === 'E') && route.screen !== 'explorer') go({ screen: 'explorer' })
      else if (e.key === 'Escape' && route.screen === 'explorer') go({ screen: prev.current })
    }
    window.addEventListener('keydown', on)
    return () => window.removeEventListener('keydown', on)
  }, [route.screen, go])

  useEffect(() => {
    const repo = summary.data?.name
    document.title = repo ? `${titles[route.screen]} · ${repo} · archdoc` : 'archdoc'
  }, [route.screen, summary.data?.name])

  if (summary.error) return <main className="boot"><Failure error={summary.error} /></main>
  if (!summary.data) return <main className="boot"><Loading /></main>
  const s = summary.data
  const version = route.v ?? null

  return (
    <RootContext.Provider value={s.root}>
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
        />
        <div className="shell-body">
          <Nav summary={s} route={route} go={go} open={navOpen} />
          {navOpen && <div className="nav-scrim" onClick={() => setNavOpen(false)} />}
          <main className={`shell-main ${route.screen === 'explorer' ? 'full' : 'page'}`}>
            {route.screen === 'overview' && <Overview summary={s} versions={versions.data ?? []} version={version} go={go} />}
            {route.screen === 'explorer' && (
              <Explorer
                version={version}
                level={route.level === 'context' ? 'context' : 'container'}
                selected={route.focus ?? null}
                onLevel={(level) => go({ screen: 'explorer', level }, { keep: true })}
                onSelect={(focus) => go({ screen: 'explorer', level: route.level, focus: focus ?? undefined }, { replace: true })}
              />
            )}
            {route.screen === 'changes' && <Changes versions={versions.data ?? []} route={route} go={go} />}
            {route.screen === 'docs' && <Documents version={version} route={route} go={go} />}
            {route.screen === 'rules' && <Corrections />}
            {route.screen === 'runs' && <NetworkRuns route={route} go={go} />}
          </main>
        </div>
      </div>
    </RootContext.Provider>
  )
}
