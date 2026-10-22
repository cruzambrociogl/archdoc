import type { ComponentType } from 'react'
import { useEffect, useMemo, useState } from 'react'
import type { HomePlace, HomeResponse } from '../api'
import type { Homes } from '../source'
import { SourceProvider } from '../source'
import { Home } from '../screens/Home'
import { Loading } from '../ui/marks'
import { ParentContext } from './parent'
import { goTo, replaceWith, usePath } from './path'
import { SystemSite } from './SystemSite'

/**
 * Home's pages (platform-surface §5.3): Home at `/`, and each place this machine remembers at
 * `/at/{id}` — a repository's pages after `#`, a system's map and repositories under it — with Home
 * first in the breadcrumb. With one place there is no Home: it opens that place.
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
  const only = data?.places.length === 1 && data.places[0].kind !== 'missing' ? data.places[0] : undefined
  const open = useMemo(
    () => (p: HomePlace, replace = false) => {
      home.opened(p.id).catch(() => {}) // remembering is a convenience; opening goes on without it
      ;(replace ? replaceWith : goTo)(`/at/${p.id}`)
    },
    [home],
  )
  useEffect(() => {
    if (!id && only) open(only, true)
  }, [id, only, open])
  const above = useMemo(() => (only ? [] : [{ label: 'Home', open: () => goTo('/') }]), [only])

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
  return <Home data={data} error={error} missing={!!id} onOpen={(p) => open(p)} onHome={() => goTo('/')} />
}
