import { useEffect, useMemo, useRef, useState } from 'react'
import { RouterProvider, createHashHistory, createRootRoute, createRoute, createRouter, redirect } from '@tanstack/react-router'
import type { CoverageResponse, Session, Summary, Version, ViewsResponse } from './api'
import { RootContext, useApi } from './api'
import { useCapabilities, useSource } from './source'
import { Failure, Loading } from './ui/marks'
import type { Screen } from './route'
import { fromOldHash, parseSearch, stringifySearch, titles, useRoute, useUnknownPath } from './route'
import { useTheme } from './theme'
import { TopBar } from './shell/TopBar'
import { Nav } from './shell/Nav'
import { Palette } from './shell/Palette'
import { Shortcuts } from './shell/Shortcuts'
import type { RailContent } from './shell/Rail'
import { MarksBand, Rail, RailProvider, marksSeen } from './shell/Rail'
import { Explorer } from './screens/explorer/Explorer'
import { Overview } from './screens/Overview'
import { Features } from './screens/Features'
import { WhatItDoes } from './screens/WhatItDoes'
import { Component } from './screens/Component'
import { Dependencies } from './screens/Dependencies'
import { WhatChanged } from './screens/WhatChanged'
import { Corrections } from './screens/Corrections'
import { Documents } from './screens/Documents'
import { NetworkRuns } from './screens/NetworkRuns'
import { Planned } from './screens/Planned'
import { Reference } from './screens/Reference'
import { About } from './screens/About'
import { Gone } from './screens/Gone'
import { AskDock } from './ui/Planned'
import { Website } from './shell/Website'
import { SystemSite } from './shell/SystemSite'
import { NoticeBand } from './shell/Notice'
import { useReconnect } from './connection'


/**
 * The app: one shell for every place (platform-surface §4.1). Where repositories are read on request
 * — the website — it opens on Start and follows a read to its documentation; for a system, on the
 * system's map; elsewhere it shows the one repository its Source reads.
 */
export function App() {
  const { runs, system } = useSource()
  if (system) return <SystemSite system={system} Repository={Repository} />
  return runs ? <Website runs={runs} Repository={Repository} /> : <Repository />
}

/**
 * One repository's pages, their address kept by a router in the hash, so the same build works at any
 * path. A link from before 22 Oct is rewritten first.
 */
function Repository() {
  const router = useMemo(() => {
    const old = fromOldHash(location.hash)
    if (old) history.replaceState(null, '', old)
    // An old link opened in a page already showing the app is caught here, before any route matches.
    const root = createRootRoute({
      component: Shell,
      beforeLoad: () => {
        const to = fromOldHash(window.location.hash)
        if (to) throw redirect({ href: to.slice(1), replace: true })
      },
    })
    const home = createRoute({ getParentRoute: () => root, path: '/', beforeLoad: () => { throw redirect({ href: '/overview' }) } })
    const any = createRoute({ getParentRoute: () => root, path: '$' })
    return createRouter({ routeTree: root.addChildren([home, any]), history: createHashHistory(), parseSearch, stringifySearch })
  }, [])
  return <RouterProvider router={router} />
}

function Shell() {
  const summary = useApi<Summary>('/api/summary')
  const versions = useApi<Version[]>('/api/versions')
  const [route, go] = useRoute()
  const unknownPath = useUnknownPath()
  const can = useCapabilities()
  useReconnect(useSource())
  const [theme, nextTheme] = useTheme()
  const [navOpen, setNavOpen] = useState(false)
  const [viewsKey, setViewsKey] = useState(0)
  const [searching, setSearching] = useState(false)
  const coverage = useApi<CoverageResponse>('/api/coverage')
  const session = useApi<Session>('/api/session')
  const [keys, setKeys] = useState(false)
  // The parts' names as the Map shows them, which the breadcrumb uses: "Inside immich-server".
  const [partNames, setPartNames] = useState<Record<string, string>>({})
  // The right rail, open on a selection; the first-visit band, until it is dismissed.
  const [rail, setRail] = useState<RailContent | null>(null)
  const [bandSeen, setBandSeen] = useState(marksSeen)
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

  if (summary.error) return <main className="boot"><Failure error={summary.error} whole /></main>
  if (!summary.data) return <main className="boot"><Loading what="the repository" /></main>
  const s = summary.data
  const version = route.v ?? null
  // A link to a version this place doesn't have says so, rather than showing another (§4.8).
  const versionGone = route.v !== undefined && !!versions.data && !versions.data.some((v) => v.id === route.v)

  return (
    <RootContext.Provider value={{ root: s.root, remote: session.data?.remote, commit: session.data?.commit }}>
      <RailProvider value={setRail}>
      <div className="shell">
        <TopBar
          summary={s}
          versions={versions.data ?? []}
          route={route}
          go={go}
          onMenu={() => setNavOpen((o) => !o)}
          theme={theme}
          onTheme={nextTheme}
          onSearch={() => setSearching(true)}
          onKeys={() => setKeys(true)}
          session={session.data}
          name={(id) => partNames[id] ?? id.replace(/^[a-z]+:/, '')}
          onMark={(mark) => setRail({ kind: 'mark', mark })}
        />
        {!bandSeen && <MarksBand onOpen={setRail} onDismiss={() => setBandSeen(true)} />}
        <div className="shell-body">
          <Nav summary={s} route={route} go={go} open={navOpen} coverage={coverage.data} />
          {navOpen && <div className="nav-scrim" onClick={() => setNavOpen(false)} />}
          <div className="main-col">
          <NoticeBand summary={s} />
          <main className={`shell-main ${route.screen === 'explorer' ? 'full' : 'page'}`}>
            {unknownPath ? (
              <Gone kind="page" address={unknownPath} go={go} />
            ) : versionGone ? (
              <Gone kind="version" v={route.v!} versions={versions.data!} name={s.name} go={go} />
            ) : (
              <>
                {route.screen === 'overview' && <Overview summary={s} versions={versions.data ?? []} version={version} go={go} coverage={coverage.data} />}
                {route.screen === 'explorer' && (
                  <Explorer version={version} versions={versions.data ?? []} route={route} go={go} editable={can.edit} views={views.data?.views ?? []} onViewsChanged={() => setViewsKey((k) => k + 1)} onNames={setPartNames} />
                )}
                {route.screen === 'features' && <WhatItDoes version={version} go={go} />}
                {route.screen === 'entries' && <Features version={version} route={route} go={go} />}
                {route.screen === 'component' && <Component version={version} route={route} go={go} />}
                {route.screen === 'deps' && <Dependencies version={version} route={route} go={go} />}
                {route.screen === 'changes' && <WhatChanged versions={versions.data ?? []} route={route} go={go} root={s.root} session={session.data} />}
                {route.screen === 'docs' && <Documents version={version} route={route} go={go} />}
                {route.screen === 'reference' && <Reference summary={s} coverage={coverage.data} go={go} />}
                {route.screen === 'about' && <About summary={s} session={session.data} go={go} />}
                {route.screen === 'plan' && <Planned route={{ ...route, focus: 'intent' }} />}
                {route.screen === 'rules' && <Corrections />}
                {route.screen === 'runs' && <NetworkRuns route={route} go={go} />}
                {route.screen === 'planned' && <Planned route={route} />}
              </>
            )}
          </main>
          {route.screen !== 'explorer' && <AskDock go={go} />}
          </div>
          {rail && <Rail content={rail} onOpen={setRail} onClose={() => setRail(null)} version={version} go={go} />}
        </div>
      </div>
      <Palette open={searching} onClose={() => setSearching(false)} go={go} views={views.data?.views ?? []} version={version} />
      {keys && <Shortcuts onClose={() => setKeys(false)} />}
      </RailProvider>
    </RootContext.Provider>
  )
}
