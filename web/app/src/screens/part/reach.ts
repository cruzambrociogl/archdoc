// One part, and two questions about it (platform-surface §17.3; app-design-screens §6): what does it
// touch, and what depends on it? Both are reach over the arrows of Apps and services — proven facts,
// followed, nothing added. Pure: the screen draws what this returns, and the tests read it.

import type { Edge, Node } from '../../api'
import type { Highlight } from '../explorer/scene'

/** Downstream — what it touches — or upstream — what depends on it. */
export type Way = 'touches' | 'depends'
export const ways: Way[] = ['touches', 'depends']

export interface Reached {
  id: string
  /** Reached through another part: the first part on the way. Absent when the arrow is direct. */
  via?: string
  /** The arrows between it and the subject, when direct: what each says, and where it is proven. */
  edges: Edge[]
}

/**
 * Every part the subject reaches (touches) or that reaches it (depends), breadth first from the
 * subject. A part reached through another is said to be reached through the first part on the way —
 * "database, through immich-server" — as the design draws it. The order is the arrows' own, so the
 * same model gives the same answer.
 */
export function reach(edges: Edge[], subject: string, way: Way): Reached[] {
  const ends = (e: Edge) => (way === 'touches' ? [e.from, e.to] : [e.to, e.from])
  const via = new Map<string, string>()
  const order: string[] = []
  const queue = [subject]
  while (queue.length) {
    const c = queue.shift()!
    for (const e of edges) {
      const [a, b] = ends(e)
      if (a !== c || b === subject || via.has(b)) continue
      via.set(b, c === subject ? '' : via.get(c) || c)
      order.push(b)
      queue.push(b)
    }
  }
  return order.map((id) => {
    const v = via.get(id)
    return v ? { id, via: v, edges: [] } : { id, edges: edges.filter((e) => (way === 'touches' ? e.from === subject && e.to === id : e.from === id && e.to === subject)) }
  })
}

/** "A", "A and B", "A, B and C". */
export function and(xs: string[]): string {
  if (xs.length <= 1) return xs[0] ?? ''
  return `${xs.slice(0, -1).join(', ')} and ${xs[xs.length - 1]}`
}

const byName = (name: (id: string) => string) => (a: string, b: string) => name(a).localeCompare(name(b)) || a.localeCompare(b)

/** The rows of the list: the direct ones first, then the rest, each by name. */
export function rows(reached: Reached[], name: (id: string) => string): Reached[] {
  const order = byName(name)
  const direct = reached.filter((r) => !r.via).sort((a, b) => order(a.id, b.id))
  const far = reached.filter((r) => r.via).sort((a, b) => order(a.id, b.id))
  return [...direct, ...far]
}

/**
 * The answer in one sentence, as the band says it:
 * "immich-mobile calls immich-server directly, and reaches database and redis through immich-server."
 * "immich-server uses database directly. immich-web and immich-mobile reach it through immich-server."
 */
export function sentence(way: Way, subject: string, reached: Reached[], name: (id: string) => string): string {
  const s = name(subject)
  const list = rows(reached, name)
  const near = list.filter((r) => !r.via).map((r) => name(r.id))
  const far = list.filter((r) => r.via)
  const vias: string[] = []
  for (const r of far) if (!vias.includes(name(r.via!))) vias.push(name(r.via!))
  const farNames = far.map((r) => name(r.id))
  if (way === 'touches') {
    if (!near.length) return `${s} calls nothing on this diagram. It only answers.`
    return `${s} calls ${and(near)} directly${farNames.length ? `, and reaches ${and(farNames)} through ${and(vias)}` : ''}.`
  }
  if (!near.length) return `Nothing on this diagram uses ${s}.`
  const first = `${and(near)} ${near.length > 1 ? 'use' : 'uses'} ${s} directly.`
  return farNames.length ? `${first} ${and(farNames)} ${farNames.length > 1 ? 'reach' : 'reaches'} it through ${and(vias)}.` : first
}

/** What the diagram shows: the subject and every part reached lit; one reached indirectly tagged with the way. */
export function highlight(subject: string, reached: Reached[], name: (id: string) => string): Highlight {
  const tags = new Map<string, string>()
  for (const r of reached) if (r.via) tags.set(r.id, `through ${name(r.via)}`)
  return { lit: new Set([subject, ...reached.map((r) => r.id)]), tags }
}

/** A part that can be asked about: anything on the diagram but a person, who is not a part of the system. */
export const askable = (n?: Node) => !!n && n.kind !== 'actor'
