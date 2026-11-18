// Every view state lives in the URL: copying the address bar reproduces the view, and back and
// forward walk through every screen, selection and comparison.
//
// The address is a path in the hash — #/architecture/containers?part=api — so the same build works
// served by archdoc at /, published under any subpath, and framed by the website. The paths are the
// link shapes of the app's design (platform-surface §17.6, settled 17 Nov): a section first, then
// what is open inside it. TanStack Router keeps the address; screens read it through useRoute and move
// with go, and never see the router.

import { useCallback, useMemo } from 'react'
import { useNavigate, useRouterState } from '@tanstack/react-router'
import { componentLevel, componentOf, dataLevel, dataOf, mainOf, structureLevel, structureOf } from './api'

export const screens = [
  'home',
  'review',
  'architecture',
  'part',
  'data',
  'features',
  'feature',
  'scenarios',
  'scenario',
  'usage',
  'changes',
  'change',
  'plan',
  'reference',
  'entries',
  'component',
  'deps',
  'docs',
  'about',
  'written',
  'report',
  'runs',
  'settings',
] as const
export type Screen = (typeof screens)[number]

export interface Route {
  screen: Screen
  /** The version on screen; absent means the latest. */
  v?: number
  /** Architecture and Data: the diagram — "context", "container", "component:<id>", "data:<id>", "dataflow". */
  level?: string
  /** Architecture and Data: the selected part; Usage: the open entry; a component page: its component. */
  focus?: string
  /** A diagram: the search, dimming what does not match. A list: its filter. */
  q?: string
  /** A diagram: dim everything not one hop from the selection. */
  dim?: string
  /** A diagram: every component of a large container, or every external system, where the default is the main ones. */
  all?: string
  /** Changes, and a diagram compared: the version compared against. */
  from?: number
  /** Documentation: the open file. */
  doc?: string
  /** What left this machine: the open run. */
  run?: number
  /** One part: which question — what it touches, or what depends on it. */
  question?: 'touches' | 'depends' | 'down'
  /** Usage: the face — how, and of which part — "api/svc:immich-server". */
  face?: string
  /** Architecture: Classic C4's blues instead of archdoc's look. */
  look?: 'classic'
  /** One scenario: how it is read — played on the parts, as a sequence, as steps. */
  way?: 'played' | 'seq' | 'steps'
  /** One scenario: the step on screen, from 1. */
  step?: number
}

// The search keys, in a fixed order so one view has one address. `focus` is spelled by what it
// is on each screen: a part on a diagram, an entry in Usage.
const order = ['v', 'part', 'entry', 'by', 'q', 'dim', 'all', 'from', 'doc', 'run', 'look', 'step'] as const
const numeric = new Set<string>(['v', 'from', 'run', 'step'])

/** The search string of a route, in the fixed order: "?part=api&q=user", or "". */
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
    if (numeric.has(k)) {
      const n = Number(raw)
      if (Number.isInteger(n)) out[k] = n
    } else {
      out[k] = raw
    }
  })
  return out
}

/** A diagram's section: what it stores and how data moves are Data; everything else is C4, in Architecture. */
export const canvasOf = (level?: string): 'architecture' | 'data' => (dataOf(level) !== undefined || level === 'dataflow' ? 'data' : 'architecture')

const enc = encodeURIComponent
const dec = (s: string) => {
  try {
    return decodeURIComponent(s)
  } catch {
    return s
  }
}

/** The address of a route: its path, then its search. */
export function hrefOf(r: Route): string {
  const search: Record<string, unknown> = { v: r.v, q: r.q, dim: r.dim, all: r.all, from: r.from, doc: r.doc, run: r.run, look: r.screen === 'architecture' ? r.look : undefined }
  let path: string
  switch (r.screen) {
    case 'architecture':
    case 'data': {
      search.part = r.focus
      const l = r.level
      const inside = componentOf(l) ?? mainOf(l) ?? structureOf(l)
      if (dataOf(l) !== undefined) path = `/data/data/${enc(dataOf(l)!)}`
      else if (l === 'dataflow') path = '/data/dataflow'
      else if (inside !== undefined) {
        path = `/architecture/components/${enc(inside)}`
        if (structureOf(l) !== undefined) search.by = 'folder'
      } else if (l === 'container') path = '/architecture/containers'
      else if (l === 'deployment') path = '/architecture/deployment'
      else if (l?.startsWith('dynamic:')) path = `/architecture/dynamic/${enc(l.slice('dynamic:'.length))}`
      else if (l === 'context') path = '/architecture/context'
      else path = r.screen === 'data' ? '/data' : '/architecture'
      break
    }
    case 'part':
      path = `/architecture/part/${enc(r.focus ?? '')}/${r.question ?? 'touches'}`
      break
    case 'usage': {
      search.entry = r.focus
      const [how, ...of] = (r.face ?? '').split('/')
      path = r.face ? `/usage/${how}/${enc(of.join('/'))}` : '/usage'
      break
    }
    case 'component':
      path = r.focus ? `/reference/component/${enc(r.focus)}` : '/reference'
      break
    case 'entries':
      search.entry = r.focus
      path = paths.entries
      break
    case 'feature':
      path = r.focus ? `/features/${enc(r.focus)}` : '/features'
      break
    case 'scenario':
      if (r.step && r.step > 1) search.step = r.step
      path = r.focus ? `/scenarios/${enc(r.focus)}/${r.way ?? 'played'}` : '/scenarios'
      break
    case 'change':
      // The version is in the path — "latest" when it is the latest — so it is not said twice.
      delete search.v
      path = `/changes/${r.v ?? 'latest'}/${enc(r.focus ?? '')}`
      break
    default:
      path = paths[r.screen]
  }
  return path + stringifySearch(search)
}

// The sections and pages whose address is fixed.
const paths: Record<Exclude<Screen, 'architecture' | 'part' | 'data' | 'usage' | 'component' | 'change' | 'feature' | 'scenario'>, string> = {
  home: '/home',
  review: '/home/review',
  features: '/features',
  scenarios: '/scenarios',
  changes: '/changes',
  plan: '/plan',
  reference: '/reference',
  entries: '/reference/entry-points',
  deps: '/reference/dependencies',
  docs: '/reference/documentation',
  about: '/about/what-was-read',
  written: '/about/what-the-ai-wrote',
  report: '/about/report',
  runs: '/about/what-left',
  settings: '/settings',
}
const byPath = new Map(Object.entries(paths).map(([s, p]) => [p, s as Screen]))

/** The route at an address, or none when no page lives there. */
export function routeAt(pathname: string, search: Record<string, unknown>): Route | undefined {
  const seg = pathname.split('/').filter(Boolean)
  const rest = (from: number) => dec(seg.slice(from).join('/'))
  const r: Route = { screen: 'home' }
  const take = <K extends keyof Route>(k: K, key: string = k) => {
    if (search[key] !== undefined) (r as unknown as Record<string, unknown>)[k] = search[key]
  }
  for (const k of ['v', 'q', 'dim', 'all', 'from', 'doc', 'run'] as const) take(k)

  const [a, b] = seg
  if (seg.length === 0) return r
  // One part: its id may hold a slash, so it is everything between "part" and the question.
  if (a === 'architecture' && b === 'part') {
    const question = seg[seg.length - 1]
    const id = rest(2).slice(0, -(question.length + 1))
    if (seg.length < 4 || (question !== 'touches' && question !== 'depends' && question !== 'down') || !id) return undefined
    r.screen = 'part'
    r.focus = id
    r.question = question
    return r
  }
  if (a === 'architecture' || a === 'data') {
    r.screen = a
    take('focus', 'part')
    if (a === 'architecture' && search.look === 'classic') r.look = 'classic'
    if (a === 'architecture') {
      if (seg.length === 1) return r
      if (b === 'context' && seg.length === 2) r.level = 'context'
      else if (b === 'containers' && seg.length === 2) r.level = 'container'
      else if (b === 'deployment' && seg.length === 2) r.level = 'deployment'
      else if (b === 'dynamic' && seg.length === 3) r.level = `dynamic:${dec(seg[2])}`
      else if (b === 'components' && seg.length > 2) r.level = search.by === 'folder' ? structureLevel(rest(2)) : componentLevel(rest(2))
      else return undefined
    } else {
      if (seg.length === 1) return r
      if (b === 'data' && seg.length > 2) r.level = dataLevel(rest(2))
      else if (b === 'dataflow' && seg.length === 2) r.level = 'dataflow'
      else return undefined
    }
    return r
  }
  // Usage, and a face: how it is used, and of which part — whose id may hold a slash.
  if (a === 'usage' && (seg.length === 1 || (seg.length >= 3 && (b === 'screens' || b === 'api' || b === 'cli')))) {
    r.screen = 'usage'
    if (seg.length >= 3) r.face = `${b}/${rest(2)}`
    take('focus', 'entry')
    return r
  }
  // One change: the version it is in, then its key — which may hold a slash.
  if (a === 'changes' && seg.length >= 3) {
    const v = b === 'latest' ? undefined : Number(b)
    if (v !== undefined && !(Number.isInteger(v) && v > 0)) return undefined
    r.screen = 'change'
    delete r.v
    if (v !== undefined) r.v = v
    r.focus = rest(2)
    return r
  }
  if (a === 'reference' && b === 'entry-points' && seg.length === 2) {
    r.screen = 'entries'
    take('focus', 'entry')
    return r
  }
  if (a === 'reference' && b === 'component' && seg.length > 2) {
    r.screen = 'component'
    r.focus = rest(2)
    return r
  }
  // One feature, by its name as an address; one scenario, by its own, then how it is read.
  if (a === 'features' && seg.length === 2) {
    r.screen = 'feature'
    r.focus = dec(b)
    return r
  }
  if (a === 'scenarios' && seg.length === 3 && (seg[2] === 'played' || seg[2] === 'seq' || seg[2] === 'steps')) {
    r.screen = 'scenario'
    r.focus = dec(b)
    r.way = seg[2]
    take('step')
    return r
  }
  if (a === 'about' && seg.length === 1) {
    r.screen = 'about'
    return r
  }
  const screen = byPath.get('/' + seg.join('/'))
  if (!screen) return undefined
  r.screen = screen
  return r
}

type Go = (next: Partial<Route>, opts?: { replace?: boolean; keep?: boolean }) => void

/** The address, when no page lives there — a link to a page that is gone (§4.8). */
export function useUnknownPath(): string | undefined {
  const location = useRouterState({ select: (s) => s.location })
  return routeAt(location.pathname, location.search as Record<string, unknown>) ? undefined : location.pathname
}

/**
 * The current route, and a way to move. `go` replaces the screen-specific state unless it is
 * passed again; the version on screen (`v`) is carried across screens. A diagram's section follows
 * its level: a move to what it stores lands in Data, whichever section asked.
 */
export function useRoute(): [Route, Go] {
  const location = useRouterState({ select: (s) => s.location })
  const navigate = useNavigate()

  const route = useMemo<Route>(
    () => routeAt(location.pathname, location.search as Record<string, unknown>) ?? { screen: 'home' },
    [location.pathname, location.search],
  )

  const go = useCallback<Go>(
    (next, opts = {}) => {
      const base: Route = opts.keep ? { ...route } : { screen: route.screen, v: route.v }
      const r = { ...base, ...next }
      if ((r.screen === 'architecture' || r.screen === 'data') && r.level) r.screen = canvasOf(r.level)
      // By href, not a typed `to`: a host may register a router of its own, and the app's paths are not its.
      navigate({ href: hrefOf(r), replace: opts.replace })
    },
    [route, navigate],
  )

  return [route, go]
}

// Plain names, single words for the sections (platform-surface §17.3): what the menu, the
// breadcrumb and the tab say.
export const titles: Record<Screen, string> = {
  home: 'Home',
  review: 'What an architect would point out',
  architecture: 'Architecture',
  part: 'One part',
  data: 'Data',
  features: 'Features',
  feature: 'A feature',
  scenarios: 'Scenarios',
  scenario: 'A scenario',
  usage: 'Usage',
  changes: 'Changes',
  change: 'One change',
  plan: 'Plan',
  reference: 'Reference',
  entries: 'Every entry point',
  component: 'A component',
  deps: 'What it relies on',
  docs: 'Documentation',
  about: 'What archdoc knows',
  written: 'What archdoc knows',
  report: 'About these docs',
  runs: 'What left this machine',
  settings: 'Settings',
}

/** The menu entry a screen belongs to: a component page is in Reference; what archdoc knows is reached from Home. */
export function sectionOf(screen: Screen): Screen {
  switch (screen) {
    case 'part':
      return 'architecture'
    case 'change':
      return 'changes'
    case 'review':
      return 'home'
    case 'feature':
      return 'features'
    case 'scenario':
      return 'scenarios'
    case 'entries':
    case 'component':
    case 'deps':
    case 'docs':
      return 'reference'
    case 'about':
    case 'written':
    case 'report':
    case 'runs':
      return 'home'
    default:
      return screen
  }
}

/** A diagram by its plain name (platform-surface §4.4): "Apps and services", not containers. */
export function levelName(level: string | undefined, name: (id: string) => string): string {
  const [kind, id] = (level ?? 'context').split(/:(.*)/s)
  switch (kind) {
    case 'context':
      return 'The outside world'
    case 'container':
      return 'Apps and services'
    case 'component':
    case 'structure':
    case 'main':
      return `Inside ${name(id)}`
    case 'data':
      return 'What it stores'
    case 'dataflow':
      return 'How data moves'
    case 'deployment':
      return 'Where it runs'
    case 'dynamic':
      return 'A scenario'
    default:
      return 'Architecture'
  }
}
