import { useEffect, useMemo, useState } from 'react'
import { RouterProvider, createHashHistory, createRootRoute, createRoute, createRouter, redirect } from '@tanstack/react-router'
import type { CoverageResponse, Session, Summary, Version, ViewsResponse } from './api'
import { RootContext, useApi } from './api'
import { useCapabilities, useSource } from './source'
import { Failure, Loading } from './ui/marks'
import { parseSearch, stringifySearch, titles, useRoute, useUnknownPath } from './route'
import { useTheme } from './theme'
import { TopBar } from './shell/TopBar'
import { Nav } from './shell/Nav'
import { Palette } from './shell/Palette'
import { Shortcuts } from './shell/Shortcuts'
import type { RailContent } from './shell/Rail'
import { MarksBand, Rail, RailProvider, marksSeen } from './shell/Rail'
import { Explorer } from './screens/explorer/Explorer'
import { Part } from './screens/part/Part'
import { Trust } from './screens/trust/Trust'
import { Usage } from './screens/usage/Usage'
import { Overview } from './screens/Overview'
import { Features } from './screens/Features'
import { WhatItDoes } from './screens/WhatItDoes'
import { Component } from './screens/Component'
import { Dependencies } from './screens/Dependencies'
import { WhatChanged } from './screens/WhatChanged'
import { Corrections } from './screens/Corrections'
import { Documents } from './screens/Documents'
import { NetworkRuns } from './screens/NetworkRuns'
import { Reference } from './screens/Reference'
import { About } from './screens/About'
import { Gone } from './screens/Gone'
import { PlanCheck } from './screens/PlanCheck'
import { Website } from './shell/Website'
import { SystemSite } from './shell/SystemSite'
import { HomeSite } from './shell/HomeSite'
import { NoticeBand } from './shell/Notice'
import { useReconnect } from './connection'


/**
 * The app: one shell for every place (platform-surface §4.1). Where repositories are read on request
 * — the website — it opens on Start and follows a read to its documentation; on this machine's Home,
 * on Home; for a system, on the system's map; elsewhere it shows the one repository its Source reads.
 */
export function App() {
  const { runs, system, home } = useSource()
  if (home) return <HomeSite home={home} Repository={Repository} />
  if (system) return <SystemSite system={system} Repository={Repository} />
  return runs ? <Website runs={runs} Repository={Repository} /> : <Repository />
}

/**
 * One repository's pages, their address kept by a router in the hash, so the same build works at any
 * path. The bare address opens Home.
 */
function Repository() {
  const router = useMemo(() => {
    const root = createRootRoute({ component: Shell })
    const home = createRoute({ getParentRoute: () => root, path: '/', beforeLoad: () => { throw redirect({ href: '/home', replace: true }) } })
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

  // A diagram fills the page: Architecture and Data.
  const canvas = route.screen === 'architecture' || route.screen === 'data'
  useEffect(() => setNavOpen(false), [route.screen])

  // The keys (platform-surface §17.6): ⌘K opens search and ask; Esc closes what is open; ? shows
  // the keys. `/` belongs to the diagram, which moves to Find a part. None works while typing.
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
      if (e.key === 'Escape' && navOpen) setNavOpen(false)
      else if (e.key === '?') setKeys(true)
    }
    window.addEventListener('keydown', on)
    return () => window.removeEventListener('keydown', on)
  }, [searching, keys, navOpen])

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
          <main className={`shell-main ${canvas || route.screen === 'part' || route.screen === 'about' || route.screen === 'written' ? 'full' : 'page'}`}>
            {unknownPath ? (
              <Gone kind="page" address={unknownPath} go={go} />
            ) : versionGone ? (
              <Gone kind="version" v={route.v!} versions={versions.data!} name={s.name} go={go} />
            ) : (
              <>
                {route.screen === 'home' && <Overview summary={s} versions={versions.data ?? []} version={version} go={go} coverage={coverage.data} />}
                {canvas && (
                  <Explorer version={version} versions={versions.data ?? []} route={route} go={go} editable={can.edit} views={views.data?.views ?? []} onViewsChanged={() => setViewsKey((k) => k + 1)} onNames={setPartNames} />
                )}
                {route.screen === 'part' && <Part route={route} go={go} version={version} onNames={setPartNames} />}
                {route.screen === 'features' && <WhatItDoes version={version} go={go} />}
                {route.screen === 'usage' && <Usage route={route} go={go} version={version} summary={s} />}
                {route.screen === 'entries' && <Features version={version} route={route} go={go} />}
                {route.screen === 'component' && <Component version={version} route={route} go={go} />}
                {route.screen === 'deps' && <Dependencies version={version} route={route} go={go} />}
                {route.screen === 'changes' && <WhatChanged versions={versions.data ?? []} route={route} go={go} root={s.root} session={session.data} />}
                {route.screen === 'docs' && <Documents version={version} route={route} go={go} />}
                {route.screen === 'reference' && <Reference summary={s} coverage={coverage.data} go={go} />}
                {(route.screen === 'about' || route.screen === 'written') && <Trust route={route} go={go} version={version} summary={s} session={session.data} onNames={setPartNames} />}
                {route.screen === 'report' && <About summary={s} session={session.data} go={go} />}
                {route.screen === 'plan' && <PlanCheck go={go} />}
                {route.screen === 'settings' && <Corrections />}
                {route.screen === 'runs' && <NetworkRuns route={route} go={go} />}
              </>
            )}
          </main>
          </div>
          {rail && <Rail content={rail} onOpen={setRail} onClose={() => setRail(null)} version={version} go={go} onSearch={() => setSearching(true)} />}
        </div>
      </div>
      <Palette open={searching} onClose={() => setSearching(false)} go={go} views={views.data?.views ?? []} version={version} onAsk={can.ask ? (question) => setRail({ kind: 'ask', question }) : undefined} />
      {keys && <Shortcuts onClose={() => setKeys(false)} />}
      </RailProvider>
    </RootContext.Provider>
  )
}
