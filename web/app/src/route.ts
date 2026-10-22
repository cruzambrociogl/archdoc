// Every view state lives in the URL (surface-spec §4.2): copying the address bar reproduces the
// view, and back and forward walk through every screen, selection and comparison.
//
// The address is a path in the hash — #/map?level=container — so the same build works served by
// archdoc at /, published under any subpath, and framed by the website. Each screen has a path in
// the information architecture of platform-surface §4: Overview, What changed, the Map, What it
// does, Reference, About these docs, Settings. TanStack Router keeps it; screens read it through
// useRoute and move with go, and never see the router.

import { useCallback, useMemo } from 'react'
import { useNavigate, useRouterState } from '@tanstack/react-router'

export const screens = [
  'overview',
  'changes',
  'plan',
  'explorer',
  'features',
  'reference',
  'component',
  'deps',
  'docs',
  'about',
  'runs',
  'rules',
  'planned',
] as const
export type Screen = (typeof screens)[number]

/** Where each screen lives (platform-surface §4.4). Its menu entry is the first segment. */
export const paths: Record<Screen, string> = {
  overview: '/overview',
  changes: '/changes',
  plan: '/plan',
  explorer: '/map',
  features: '/does',
  reference: '/reference',
  component: '/reference/component',
  deps: '/reference/dependencies',
  docs: '/reference/documentation',
  about: '/about',
  runs: '/about/what-left',
  rules: '/settings',
  planned: '/not-built-yet',
}
const byPath = new Map(Object.entries(paths).map(([s, p]) => [p, s as Screen]))

export interface Route {
  screen: Screen
  /** The version on screen; absent means the latest. */
  v?: number
  /** Map: the level. */
  level?: string
  /** Map: the selected element; a component page's component; Not built yet: the plan opened. */
  focus?: string
  /** Map: the search, dimming what does not match. */
  q?: string
  /** Map: dim everything not one hop from the selection. */
  dim?: string
  /** Map: every component of a large container, where the default is its main ones. */
  all?: string
  /** What changed: the version compared against. */
  from?: number
  /** Documentation: the open file. */
  doc?: string
  /** What left this machine: the open run. */
  run?: number
}

// Fixed key order, so the same view always has the same address.
const order: (keyof Route)[] = ['v', 'level', 'focus', 'q', 'dim', 'all', 'from', 'doc', 'run']
const numeric = new Set<keyof Route>(['v', 'from', 'run'])

/** The search string of a route, in the fixed order: "?level=container&focus=api", or "". */
export function stringifySearch(search: Record<string, unknown>): string {
  const q = new URLSearchParams()
  for (const k of order) {
    const v = search[k]
    if (v !== undefined && v !== null && v !== '') q.set(k, String(v))
  }
  const s = q.toString()
  return s ? `?${s}` : ''
}

/** A search string's values, numbers where the route has numbers. */
export function parseSearch(search: string): Record<string, unknown> {
  const out: Record<string, unknown> = {}
  new URLSearchParams(search.replace(/^\?/, '')).forEach((raw, k) => {
    if (raw === '') return
    if (numeric.has(k as keyof Route)) {
      const n = Number(raw)
      if (Number.isInteger(n)) out[k] = n
    } else {
      out[k] = raw
    }
  })
  return out
}

/** The screen at a path, or none. */
export const screenAt = (pathname: string): Screen | undefined => byPath.get(pathname.replace(/\/+$/, '') || '/')

/**
 * The address an old link meant. Before 22 Oct a view was #screen=explorer&level=…; published sites
 * and bookmarks still carry those, so they open where that screen lives now (platform-surface §4.8,
 * "a page that is gone").
 */
export function fromOldHash(hash: string): string | undefined {
  const q = new URLSearchParams(hash.replace(/^#/, ''))
  const old = q.get('screen')
  if (!old) return undefined
  const moved: Record<string, Screen> = { coverage: 'about' }
  const screen: Screen = moved[old] ?? ((screens as readonly string[]).includes(old) ? (old as Screen) : 'overview')
  q.delete('screen')
  return `#${paths[screen]}${stringifySearch(parseSearch(q.toString()))}`
}

type Go = (next: Partial<Route>, opts?: { replace?: boolean; keep?: boolean }) => void

/**
 * The current route, and a way to move. `go` replaces the screen-specific state unless it is
 * passed again; the version on screen (`v`) is carried across screens.
 */
export function useRoute(): [Route, Go] {
  const location = useRouterState({ select: (s) => s.location })
  const navigate = useNavigate()

  const route = useMemo<Route>(() => {
    const search = location.search as Record<string, unknown>
    const r: Route = { screen: screenAt(location.pathname) ?? 'overview' }
    for (const k of order) {
      const v = search[k]
      if (v !== undefined) (r as unknown as Record<string, unknown>)[k] = v
    }
    return r
  }, [location.pathname, location.search])

  const go = useCallback<Go>(
    (next, opts = {}) => {
      const base: Route = opts.keep ? { ...route } : { screen: route.screen, v: route.v }
      const r = { ...base, ...next }
      const search: Record<string, unknown> = {}
      for (const k of order) if (r[k] !== undefined && r[k] !== '') search[k] = r[k]
      navigate({ to: paths[r.screen], search, replace: opts.replace })
    },
    [route, navigate],
  )

  return [route, go]
}
