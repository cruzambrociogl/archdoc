import type { ComponentType } from 'react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import type { SystemResponse } from '../api'
import type { Systems } from '../source'
import { SourceProvider } from '../source'
import { SystemMap } from '../screens/SystemMap'
import { ParentContext } from './parent'
import { Loading } from '../ui/marks'

/** Where in a system: its map (`/`), or one repository (`/repos/{name}`), whose own pages follow `#`. */
type Where = { page: 'map' } | { page: 'repo'; name: string }

function where(pathname = location.pathname): Where {
  const m = pathname.match(/^\/repos\/([^/]+)\/?$/)
  return m ? { page: 'repo', name: decodeURIComponent(m[1]) } : { page: 'map' }
}

/**
 * A system's pages (platform-surface §4.3, §4.6): its map, and each repository's pages through the
 * Source the system opens for it — under the system in the breadcrumb, which leads back to the map.
 */
export function SystemSite({ system, Repository }: { system: Systems; Repository: ComponentType }) {
  const [at, setAt] = useState(where)
  const [sys, setSys] = useState<SystemResponse>()
  const [error, setError] = useState<Error>()
  useEffect(() => {
    const back = () => setAt(where())
    window.addEventListener('popstate', back)
    return () => window.removeEventListener('popstate', back)
  }, [])
  // The page changes first, the address after: a repository's router, still listening while it is
  // on screen, would otherwise take the new address for one of its own and add its page to it.
  const [moving, setMoving] = useState<string>()
  const open = useCallback((path: string) => {
    setAt(where(path))
    setMoving(path)
  }, [])
  useEffect(() => {
    if (moving === undefined) return
    history.pushState(null, '', moving)
    setMoving(undefined)
  }, [moving])
  const toMap = useCallback(() => open('/'), [open])

  // Read again on coming back to the map: a repository documented meanwhile shows its new version.
  const need = at.page === 'map' || !sys
  useEffect(() => {
    if (need) system.get().then(setSys, (e: Error) => setError(e))
  }, [system, need])

  const name = at.page === 'repo' ? at.name : undefined
  const source = useMemo(() => (name ? system.open(name) : undefined), [system, name])
  const parent = useMemo(() => ({ label: sys?.name ?? '', repo: name ?? '', open: toMap }), [sys?.name, name, toMap])
  const missing = name && sys && !sys.repos.some((r) => r.name === name) ? name : undefined

  // A repository waits for its system, so a name the system doesn't hold is told, never asked for.
  if (source && !sys && !error) {
    return (
      <main className="boot">
        <Loading what="the system" />
      </main>
    )
  }
  if (source && !missing) {
    return (
      <SourceProvider value={source}>
        <ParentContext.Provider value={parent}>
          <Repository key={name} />
        </ParentContext.Provider>
      </SourceProvider>
    )
  }
  return <SystemMap sys={sys} error={error} missing={missing} onOpen={(r) => open(`/repos/${encodeURIComponent(r)}`)} onSystem={toMap} />
}
