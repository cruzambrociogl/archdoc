// The structure sentence (platform-surface §6.1): what a system is made of and how its parts call
// each other, written from the container graph by a template — proven, never interpreted, and the
// only paragraph a repository gets when no one has written its purpose. Every part it names is
// returned with it, so the screen can link each one to the Map and cite it.

import type { Model, Node, SystemLink, SystemResponse } from './api'

/** A run of text, or a part named in it. */
export type Piece = string | { node: Node }

const counted = ['no', 'one', 'two', 'three', 'four', 'five', 'six', 'seven', 'eight', 'nine', 'ten', 'eleven', 'twelve']
export const number = (n: number) => counted[n] ?? String(n)
const capital = (s: string) => s.charAt(0).toUpperCase() + s.slice(1)

/** "a", "a and b", "a, b and c" — as pieces, each part a node. */
function list(nodes: Node[]): Piece[] {
  const out: Piece[] = []
  nodes.forEach((n, i) => {
    if (i > 0) out.push(i === nodes.length - 1 ? ' and ' : ', ')
    out.push({ node: n })
  })
  return out
}

/** Names up to three parts, and counts more: "Five parts". */
function named(nodes: Node[], noun: string): Piece[] {
  return nodes.length <= 3 ? list(nodes) : [`${capital(number(nodes.length))} ${noun}`]
}

/**
 * The sentences, as pieces. `name` is the system's. With one part the system is that part; with a hub
 * — the application most others call — the sentence follows the calls into it and out of it; with
 * none, it lists what there is.
 */
export function structureSentence(name: string, view: Model): Piece[][] {
  const nodes = view.nodes ?? []
  const edges = (view.edges ?? []).filter((e) => e.from !== e.to)
  const by = new Map(nodes.map((n) => [n.id, n]))
  const apps = nodes.filter((n) => n.kind === 'application')
  const stores = nodes.filter((n) => n.kind === 'datastore')
  const queues = nodes.filter((n) => n.kind === 'queue')
  const outside = nodes.filter((n) => n.kind === 'external')
  const parts = [...apps, ...stores, ...queues]
  if (parts.length === 0) return []

  const sentences: Piece[][] = []
  sentences.push(parts.length === 1 ? [`${name} is one part: `, { node: parts[0] }, '.'] : [`${name} is made of ${number(parts.length)} parts.`])

  const into = (id: string) => edges.filter((e) => e.to === id && by.get(e.from)?.kind === 'application')
  const outOf = (id: string, kind: string) => uniq(edges.filter((e) => e.from === id).map((e) => by.get(e.to)).filter((n): n is Node => n?.kind === kind))
  const hub = [...apps].sort((a, b) => into(b.id).length - into(a.id).length || a.name.localeCompare(b.name))[0]
  const callers = hub ? uniq(into(hub.id).map((e) => by.get(e.from)!)).filter((n) => n.id !== hub.id) : []

  if (hub && callers.length >= 1) {
    const does: Piece[][] = []
    const s = outOf(hub.id, 'datastore')
    const q = outOf(hub.id, 'queue')
    const a = outOf(hub.id, 'application')
    if (s.length) does.push(['stores data in ', ...list(s)])
    if (q.length) does.push(['queues work in ', ...list(q)])
    if (a.length) does.push(['calls ', ...list(a)])
    const s1: Piece[] = [...named(callers, 'parts'), callers.length === 1 ? ' calls ' : ' call ', { node: hub }]
    does.forEach((d, i) => s1.push(i === 0 ? ', which ' : i === does.length - 1 ? ' and ' : ', ', ...d))
    s1.push('.')
    sentences.push(s1)
  } else if (parts.length > 1) {
    const s1: Piece[] = []
    if (apps.length) s1.push(apps.length === 1 ? 'Its application is ' : 'Its applications are ', ...named(apps, 'applications'))
    const kept = [...stores, ...queues]
    if (kept.length) s1.push(apps.length ? '; it keeps data in ' : 'It keeps data in ', ...named(kept, 'stores'))
    s1.push('.')
    sentences.push(s1)
  }

  if (outside.length) sentences.push(['It reaches ', ...(outside.length <= 2 ? list(outside) : [`${number(outside.length)} outside services`]), '.'])
  return sentences
}

/** Every part a sentence names, once, in order — what the Evidence list cites. */
export function namedIn(sentences: Piece[][]): Node[] {
  return uniq(sentences.flat().flatMap((p) => (typeof p === 'string' ? [] : [p.node])))
}

function uniq(nodes: Node[]): Node[] {
  const seen = new Set<string>()
  return nodes.filter((n) => !seen.has(n.id) && seen.add(n.id))
}


/** "a", "a and b", "a, b and c". */
const and = (items: string[]) => (items.length <= 1 ? items.join('') : `${items.slice(0, -1).join(', ')} and ${items[items.length - 1]}`)
const plural = (n: number, one: string, many: string) => `${number(n)} ${n === 1 ? one : many}`

/** Which of the other's code a link uses, in words — and how far the other has moved since (§4.6, drift). */
export function pinWords(l: SystemLink): string {
  const p = l.pin
  if (!p) return l.version ? ` (${l.version})` : ''
  if (p.kind === 'checkout') return ', from the checkout beside it'
  const ref = p.kind === 'commit' ? p.ref.slice(0, 7) : p.ref
  if (p.ahead > 0) return ` at ${ref}, which ${l.to} is ${plural(p.ahead, 'commit', 'commits')} past`
  if (p.ahead === 0) return ` at ${ref}, the commit ${l.to}'s docs are from`
  return ` at ${ref}`
}

/**
 * The system sentence (platform-surface §4.6): how many repositories, then, for each one that
 * requires another, which of its packages and at what — written by a template from the links, each
 * of which is cited at both ends, so it is proven. What nothing links says so.
 */
export function systemSentence(sys: SystemResponse): string[] {
  const out = [`${sys.name} is ${plural(sys.repos.length, 'repository', 'repositories')}: ${and(sys.repos.map((r) => r.name))}.`]
  const pairs = new Map<string, SystemLink[]>()
  for (const l of sys.links) {
    const k = `${l.from}\u0000${l.to}`
    pairs.set(k, [...(pairs.get(k) ?? []), l])
  }
  for (const links of pairs.values()) {
    const { from, to } = links[0]
    const items = links.map((l) => l.package + pinWords(l))
    out.push(
      links.length === 1
        ? `${from} requires ${to}'s ${items[0]}.`
        : links.length <= 3
          ? `${from} requires ${number(links.length)} of ${to}'s packages: ${items.slice(0, -1).join('; ')}; and ${items[items.length - 1]}.`
          : `${from} requires ${number(links.length)} of ${to}'s packages.`,
    )
  }
  const linked = new Set(sys.links.flatMap((l) => [l.from, l.to]))
  const alone = sys.repos.filter((r) => !linked.has(r.name)).map((r) => r.name)
  if (sys.links.length === 0) out.push('archdoc found nothing linking them that it could prove.')
  else if (alone.length) out.push(`${and(alone)} ${alone.length === 1 ? 'is' : 'are'} linked to none of the others by anything archdoc could prove.`)
  return out
}
