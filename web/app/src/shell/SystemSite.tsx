import type { ComponentType } from 'react'
import { useEffect, useMemo, useState } from 'react'
import type { SystemResponse } from '../api'
import type { Systems } from '../source'
import { SourceProvider } from '../source'
import { SystemMap } from '../screens/SystemMap'
import { SystemChanges } from '../screens/SystemChanges'
import { Loading } from '../ui/marks'
import type { Crumb } from './parent'
import { ParentContext } from './parent'
import { goTo, usePath } from './path'

/**
 * A system's pages (platform-surface §4.3, §4.6): its map at `base`, what changed across it at
 * `base/changes`, and each repository's pages at
 * `base/repos/{name}`, after `#`, through the Source the system opens for it — under the system in
 * the breadcrumb, and under whatever is above the system (`above`: Home).
 */
export function SystemSite({ system, Repository, base = '', above = [] }: { system: Systems; Repository: ComponentType; base?: string; above?: Crumb[] }) {
  const path = usePath()
  const rest = path.slice(base.length)
  const m = rest.match(/^\/repos\/([^/]+)\/?$/)
  const name = m ? decodeURIComponent(m[1]) : undefined
  const changes = /^\/changes\/?$/.test(rest)
  const [sys, setSys] = useState<SystemResponse>()
  const [error, setError] = useState<Error>()

  // Read again on coming back to the map: a repository documented meanwhile shows its new version.
  const need = !name || !sys
  useEffect(() => {
    if (need) system.get().then(setSys, (e: Error) => setError(e))
  }, [system, need])

  const source = useMemo(() => (name ? system.open(name) : undefined), [system, name])
  const toMap = useMemo(() => () => goTo(base || '/'), [base])
  const parent = useMemo(() => ({ crumbs: [...above, { label: sys?.name ?? '', open: toMap }], repo: name ?? '' }), [above, sys?.name, name, toMap])
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
  const toChanges = () => goTo(`${base}/changes`)
  const open = (r: string, page?: string) => goTo(`${base}/repos/${encodeURIComponent(r)}${page === 'changes' ? '#/changes' : ''}`)
  if (changes) return <SystemChanges sys={sys} error={error} above={above} system={system} onMap={toMap} onChanges={toChanges} onOpen={open} />
  return <SystemMap sys={sys} error={error} missing={missing} above={above} onOpen={open} onSystem={toMap} onChanges={toChanges} />
}
