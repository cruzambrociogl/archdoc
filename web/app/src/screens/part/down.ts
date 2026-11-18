// What breaks without one part (One part's third question; settled points 4 and 5): what the engine
// worked out (/api/breaks), said in sentences and drawn with the highlight. Pure: the screen draws
// what this returns, and the tests read it.

import type { Highlight } from '../explorer/scene'
import { and } from './reach'

export interface Effect {
  entry: string
  why: string
  cite: { file: string; line: number }
}
export interface Dependent {
  id: string
  how: string
  known: boolean
  stops: Effect[]
  unaffected: number
  unfollowed: number
  provenance: { file: string; line: number }[] | null
}
export interface Breakage {
  part: string
  dependents: Dependent[]
  through: { id: string; via: string }[]
  basis: string
}

const kindOf = (entry: string) => entry.split(':')[0]
const nouns: Record<string, [string, string]> = {
  route: ['route', 'routes'],
  page: ['page', 'pages'],
  command: ['command', 'commands'],
  job: ['background job', 'background jobs'],
}
const count = (n: number, kind: string) => `${n} ${(nouns[kind] ?? ['way in', 'ways in'])[n === 1 ? 0 : 1]}`

const rank = (entry: string) => {
  const i = ['route', 'page', 'command', 'job'].indexOf(kindOf(entry))
  return i < 0 ? 4 : i
}

/** What stops, as the list shows it: what people wait on first — routes, pages, commands — then background jobs. */
export function ordered(stops: Effect[]): Effect[] {
  return [...stops].sort((a, b) => rank(a.entry) - rank(b.entry) || a.entry.localeCompare(b.entry))
}

/** "240 routes and 62 background jobs": what stops, by kind, in a fixed order. */
export function stopsByKind(stops: Effect[]): string {
  const by = new Map<string, number>()
  for (const s of stops) by.set(kindOf(s.entry), (by.get(kindOf(s.entry)) ?? 0) + 1)
  return and(['route', 'page', 'command', 'job', ...[...by.keys()].filter((k) => !nouns[k]).sort()].filter((k) => by.has(k)).map((k) => count(by.get(k)!, k)))
}

/**
 * The answer in sentences, as the band says it:
 * "immich-server stops 240 routes and 62 background jobs that read or write its tables. 61 of its
 * ways in reach nothing of it; 17 weren't followed, so can't be told."
 */
export function downSentence(b: Breakage, name: (id: string) => string): string {
  const parts = b.dependents.filter((d) => !d.how.startsWith('a person'))
  if (!parts.length) return `Nothing on this diagram uses ${name(b.part)}, so nothing here breaks without it.`
  const out: string[] = []
  for (const d of parts.filter((d) => d.known)) {
    const why = d.how === 'queues its jobs on it' ? 'that queue work on it, or are taken from its queue' : 'that read or write its tables'
    out.push(d.stops.length ? `${name(d.id)} stops ${stopsByKind(d.stops)} ${why}.` : `${name(d.id)} keeps working: none of its ways in ${d.how === 'queues its jobs on it' ? 'queue work on it' : 'read or write its tables'}.`)
    const rest = [d.unaffected ? `${d.unaffected} of its ways in reach nothing of it` : '', d.unfollowed ? `${d.unfollowed} weren’t followed, so can’t be told` : ''].filter(Boolean)
    if (rest.length && d.stops.length) out.push(`${rest.join('; ')}.`.replace(/^./, (c) => c.toUpperCase()))
  }
  const unknown = parts.filter((d) => !d.known)
  if (unknown.length)
    out.push(`${and(unknown.map((d) => name(d.id)))} ${unknown.length === 1 ? 'waits' : 'wait'} for its answer, but which of ${unknown.length === 1 ? 'its' : 'their'} ways in make the call isn’t recorded.`)
  return out.join(' ')
}

/** The diagram: the part down — hatched, tagged — what depends on it tagged, the rest dimmed. */
export function downHighlight(b: Breakage): Highlight {
  const tags = new Map<string, string>([[b.part, 'Down']])
  for (const d of b.dependents) if (!d.how.startsWith('a person')) tags.set(d.id, 'depends on it')
  return {
    lit: new Set([b.part, ...b.dependents.map((d) => d.id), ...b.through.map((t) => t.id)]),
    tags,
    tone: 'ink',
    hatched: new Set([b.part]),
  }
}
