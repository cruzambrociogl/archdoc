// Every view state lives in the URL (surface-spec §4.2): copying the address bar reproduces the
// view, and back and forward walk through every screen, selection and comparison.

import { useCallback, useEffect, useState } from 'react'

export const screens = ['overview', 'explorer', 'component', 'features', 'deps', 'changes', 'docs', 'coverage', 'rules', 'runs', 'planned'] as const
export type Screen = (typeof screens)[number]

export interface Route {
  screen: Screen
  /** The version on screen; absent means the latest. */
  v?: number
  /** Explorer: the C4 level. */
  level?: string
  /** Explorer: the selected element. */
  focus?: string
  /** Explorer: the search, dimming what does not match. */
  q?: string
  /** Explorer: dim everything not one hop from the selection. */
  dim?: string
  /** Explorer: every component of a large container, where the default is its main ones. */
  all?: string
  /** Changes: the version compared against. */
  from?: number
  /** Documents: the open file. */
  doc?: string
  /** Network runs: the open run. */
  run?: number
}

// Fixed key order, so the same view always has the same address.
const order: (keyof Route)[] = ['screen', 'v', 'level', 'focus', 'q', 'dim', 'all', 'from', 'doc', 'run']
const numeric = new Set<keyof Route>(['v', 'from', 'run'])

export function parse(hash: string): Route {
  const q = new URLSearchParams(hash.replace(/^#/, ''))
  const screen = q.get('screen') as Screen
  const r: Route = { screen: screens.includes(screen) ? screen : 'overview' }
  for (const k of order) {
    if (k === 'screen') continue
    const raw = q.get(k)
    if (raw === null || raw === '') continue
    if (numeric.has(k)) {
      const n = Number(raw)
      if (Number.isInteger(n)) (r as unknown as Record<string, number>)[k] = n
    } else {
      ;(r as unknown as Record<string, string>)[k] = raw
    }
  }
  return r
}

export function format(r: Route): string {
  const q = new URLSearchParams()
  for (const k of order) {
    const v = r[k]
    if (v !== undefined && v !== null && v !== '') q.set(k, String(v))
  }
  return '#' + q.toString()
}

/**
 * The current route, and a way to move. `go` replaces the screen-specific state unless it is
 * passed again; the version on screen (`v`) is carried across screens.
 */
export function useRoute(): [Route, (next: Partial<Route>, opts?: { replace?: boolean; keep?: boolean }) => void] {
  const [route, setRoute] = useState(() => parse(location.hash))

  useEffect(() => {
    const on = () => setRoute(parse(location.hash))
    window.addEventListener('hashchange', on)
    if (!location.hash) history.replaceState(null, '', format(route))
    return () => window.removeEventListener('hashchange', on)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const go = useCallback((next: Partial<Route>, opts: { replace?: boolean; keep?: boolean } = {}) => {
    setRoute((cur) => {
      const base: Route = opts.keep ? { ...cur } : { screen: cur.screen, v: cur.v }
      const r = { ...base, ...next }
      const h = format(r)
      if (h !== location.hash) {
        if (opts.replace) history.replaceState(null, '', h)
        else history.pushState(null, '', h)
      }
      return r
    })
  }, [])

  return [route, go]
}
