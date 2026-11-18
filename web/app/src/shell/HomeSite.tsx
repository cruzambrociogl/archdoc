import type { ComponentType } from 'react'
import { useEffect, useMemo, useState } from 'react'
import type { HomePlace, HomeResponse } from '../api'
import type { Homes } from '../source'
import { SourceProvider } from '../source'
import { Library } from '../screens/Library'
import { Loading } from '../ui/marks'
import { ParentContext } from './parent'
import { goTo, replaceWith, usePath } from './path'
import { SystemSite } from './SystemSite'

/**
 * The library's pages (platform-surface §5.3, Home there): the library at `/`, and each place this
 * machine remembers at `/at/{id}` — a repository's pages after `#`, a system's map and repositories
 * under it — with the library first in the breadcrumb. It is there with one place too: this
 * machine's one archdoc serves every project, and the library is where the next one appears
 * (decisions, 18 Nov 2026).
 */
export function HomeSite({ home, Repository }: { home: Homes; Repository: ComponentType }) {
  const path = usePath()
  const id = path.match(/^\/at\/([^/]+)/)?.[1]
  const [data, setData] = useState<HomeResponse>()
  const [error, setError] = useState<Error>()

  // Read again on coming back to Home: what was just opened is no longer "changed".
  const need = !id || !data
  useEffect(() => {
    if (need) home.get().then(setData, (e: Error) => setError(e))
  }, [home, need])

  const place = data?.places.find((p) => p.id === id)
  const source = useMemo(() => (place && place.kind !== 'missing' ? home.open(place) : undefined), [home, place])
  const open = useMemo(
    () => (p: HomePlace, replace = false) => {
      home.opened(p.id).catch(() => {}) // remembering is a convenience; opening goes on without it
      ;(replace ? replaceWith : goTo)(`/at/${p.id}`)
    },
    [home],
  )
  const above = useMemo(() => [{ label: 'Library', open: () => goTo('/') }], [])

  if (!data && !error) {
    return (
      <main className="boot">
        <Loading what="the places this machine remembers" />
      </main>
    )
  }
  if (place && source) {
    return (
      <SourceProvider value={source}>
        {place.kind === 'system' && source.system ? (
          <SystemSite system={source.system} Repository={Repository} base={`/at/${place.id}`} above={above} />
        ) : (
          <ParentContext.Provider value={{ crumbs: above, repo: place.name }}>
            <Repository key={place.id} />
          </ParentContext.Provider>
        )}
      </SourceProvider>
    )
  }
  return <Library data={data} error={error} missing={!!id} onOpen={(p) => open(p)} onHome={() => goTo('/')} />
}
