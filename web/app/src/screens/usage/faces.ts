// Usage (platform-surface §17.3; app-design-screens §10): how it is used — on screens, through an
// API, from a command line — one face for each way and each part it is used in. Every value is the
// model's: the entries the code declares, and what each sets off as the engine followed it. Pure: the
// screen draws what this returns, and the tests read it.

import type { Entry, Flow } from '../../api'

/** How a face is used: its pages, its HTTP routes, its commands. Background jobs are not a way people use it. */
export type How = 'screens' | 'api' | 'cli'
const order: How[] = ['screens', 'api', 'cli']

export const howOf = (e: Entry): How | undefined => (e.kind === 'page' ? 'screens' : e.kind === 'http' ? 'api' : e.kind === 'command' ? 'cli' : undefined)

export interface Face {
  /** "api/svc:immich-server": how, and of which part — the face's address. */
  key: string
  how: How
  container: string
  entries: Entry[]
}

export const faceKey = (how: How, container: string) => `${how}/${container}`
export const faceOfEntry = (e: Entry) => {
  const how = howOf(e)
  return how ? faceKey(how, e.container) : undefined
}

/**
 * The faces, in a fixed order: screens, then APIs, then command lines; within each, the part with the
 * most entries first, then by id. Each face's entries in the order of their addresses.
 */
export function facesOf(entries: Entry[]): Face[] {
  const by = new Map<string, Face>()
  for (const e of entries) {
    const how = howOf(e)
    if (!how) continue
    const key = faceKey(how, e.container)
    if (!by.has(key)) by.set(key, { key, how, container: e.container, entries: [] })
    by.get(key)!.entries.push(e)
  }
  const faces = [...by.values()]
  for (const f of faces) f.entries.sort((a, b) => addressOf(a).localeCompare(addressOf(b)) || a.id.localeCompare(b.id))
  return faces.sort((a, b) => order.indexOf(a.how) - order.indexOf(b.how) || b.entries.length - a.entries.length || a.container.localeCompare(b.container))
}

/** The entry's address, as a reader would type it: "/photos", "POST /api/assets", "upload". */
export function addressOf(e: Entry): string {
  return e.kind === 'http' ? `${e.method.toUpperCase()} ${e.path}` : e.path
}

/** Its plain name: what its own code says it does, where the code says it; else its address. */
export function nameOf(e: Entry): string {
  return e.summary?.trim() || addressOf(e)
}

/** "A", "A and B", "A, B and C". */
export function and(xs: string[]): string {
  if (xs.length <= 1) return xs[0] ?? ''
  return `${xs.slice(0, -1).join(', ')} and ${xs[xs.length - 1]}`
}

const nouns: Record<How, [string, string]> = { screens: ['page', 'pages'], api: ['route', 'routes'], cli: ['command', 'commands'] }
export const countOf = (f: Face) => `${f.entries.length} ${nouns[f.how][f.entries.length === 1 ? 0 : 1]}`

/** The face's name, as its tab says it. */
export const faceName: Record<How, string> = { screens: 'Screens', api: 'API', cli: 'Command line' }
/** What its list is headed: what someone can do there. */
export const faceHeading: Record<How, string> = { screens: 'What people open', api: 'What a client can do', cli: 'What you can run from a terminal' }

/**
 * The sentence naming the faces:
 * "immich is used 5 ways: on the screens of immich-web, through the APIs of immich-server and
 * immich-machine-learning, and from the command lines of immich-server and @immich/cli."
 */
export function usedSentence(system: string, faces: Face[], name: (id: string) => string): string {
  if (!faces.length) return `archdoc found no pages, routes or commands in ${system}’s code.`
  const of = (how: How) => faces.filter((f) => f.how === how).map((f) => name(f.container))
  const parts: string[] = []
  const screens = of('screens'), apis = of('api'), clis = of('cli')
  if (screens.length) parts.push(`on the screens of ${and(screens)}`)
  if (apis.length) parts.push(`through the ${apis.length === 1 ? 'API' : 'APIs'} of ${and(apis)}`)
  if (clis.length) parts.push(`from the ${clis.length === 1 ? 'command line' : 'command lines'} of ${and(clis)}`)
  const ways = faces.length === 1 ? 'one way' : `${faces.length} ways`
  // Each part may hold an "and" of its own, so three take a comma before the last.
  const said = parts.length > 2 ? `${parts.slice(0, -1).join(', ')}, and ${parts[parts.length - 1]}` : and(parts)
  return `${system} is used ${ways}: ${said}.`
}

/** What its list is made from, as the face's mark says it: proven, from the code, nothing by AI. */
export function faceMark(f: Face, flows: Flow[]): string {
  if (f.how === 'screens') {
    const followed = new Set(flows.map((x) => x.entry))
    const calls = f.entries.some((e) => followed.has(e.id))
    return `From the code: each page’s path and the file that draws it.${calls ? '' : ' archdoc doesn’t read what a page shows or calls yet.'} Nothing here was written by AI.`
  }
  if (f.how === 'api') return 'From the code: each route’s method, path and handler, and its summary where the code states one. Nothing here was written by AI.'
  return 'From the code: each command’s name and handler, and its description where the code states one. Nothing here was written by AI.'
}

/** An entry matches a filter by its name, address or handler. */
export function matches(e: Entry, q: string): boolean {
  const t = q.trim().toLowerCase()
  return !t || nameOf(e).toLowerCase().includes(t) || addressOf(e).toLowerCase().includes(t) || e.handler.toLowerCase().includes(t)
}

/** What an entry sets off, as the engine followed it: the tables it reads and changes, the jobs it queues, the calls it cannot follow. */
export interface SetsOff {
  reads: string[]
  writes: string[]
  updates: string[]
  deletes: string[]
  queues: string[]
  /** Calls to an address set at run time. */
  loose: number
  /** The flow stopped early: what is listed is what came before the cut. */
  cut: boolean
}

export function setsOff(flow?: Flow): SetsOff | undefined {
  if (!flow) return undefined
  const kinds = new Map(flow.participants.map((p) => [p.id, p]))
  const out: SetsOff = { reads: [], writes: [], updates: [], deletes: [], queues: [], loose: 0, cut: !!flow.cut }
  const add = (xs: string[], x: string) => xs.includes(x) || xs.push(x)
  for (const s of flow.steps) {
    const p = kinds.get(s.to)
    if (!p) continue
    if (p.kind === 'table') {
      if (s.call === 'reads') add(out.reads, p.name)
      else if (s.call === 'writes') add(out.writes, p.name)
      else if (s.call === 'updates') add(out.updates, p.name)
      else if (s.call === 'deletes') add(out.deletes, p.name)
    } else if (p.kind === 'job') add(out.queues, p.name)
    else if (p.kind === 'unresolved') out.loose++
  }
  return out
}

/** What it sets off, in one sentence: "Reads asset and user; writes asset_exif; queues SidecarWrite." */
export function setsOffSentence(s: SetsOff | undefined): string {
  if (!s) return 'archdoc didn’t follow what it does.'
  const parts: string[] = []
  const list = (verb: string, xs: string[]) => xs.length && parts.push(`${verb} ${and(xs)}`)
  list('reads', s.reads)
  list('writes', s.writes)
  list('updates', s.updates)
  list('deletes', s.deletes)
  list('queues', s.queues)
  if (s.loose) parts.push(`makes ${s.loose === 1 ? 'a call' : `${s.loose} calls`} to an address set at run time`)
  if (!parts.length) return s.cut ? 'Nothing archdoc could follow before it stopped.' : 'No tables, jobs or calls archdoc could follow.'
  const said = parts.join('; ')
  return `${said.charAt(0).toUpperCase()}${said.slice(1)}${s.cut ? '; and more, past where archdoc stopped following' : ''}.`
}
