// The system explained, as the app reads it: where each claim's citations lead, which boxes of Apps
// and services a feature or a scenario lights, and how a scenario's steps divide between what happens
// while someone waits and what happens later. Nothing here is computed beyond following the IDs the
// explanation stored to the facts they name (the app's first rule: it computes nothing the engine did
// not store).

import type { Entry, Interpretation, InterpretedGroup, Model, Node, Scenario } from './api'
import type { Route } from './route'

/** A name as an address: "Face and people recognition" → "face-and-people-recognition". */
export function slug(name: string): string {
  return name
    .toLowerCase()
    .normalize('NFKD')
    .replace(/[̀-ͯ]/g, '')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
}

/** Every item's address, unique: a second "Search" becomes "search-2". */
function slugs<T>(items: T[], name: (t: T) => string): string[] {
  const seen = new Map<string, number>()
  return items.map((t) => {
    const s = slug(name(t)) || 'untitled'
    const n = (seen.get(s) ?? 0) + 1
    seen.set(s, n)
    return n === 1 ? s : `${s}-${n}`
  })
}

export const featureSlugs = (it: Interpretation) => slugs(it.features ?? [], (f) => f.name)
export const scenarioSlugs = (it: Interpretation) => slugs(it.scenarios ?? [], (s) => s.title)

/** A feature by its address, with its index — feature:N in a citation is the Nth, counting from 1. */
export function featureAt(it: Interpretation, at: string): { feature: InterpretedGroup; index: number } | undefined {
  const i = featureSlugs(it).indexOf(at)
  return i < 0 ? undefined : { feature: it.features[i], index: i }
}

export function scenarioAt(it: Interpretation, at: string): { scenario: Scenario; index: number } | undefined {
  const all = it.scenarios ?? []
  const i = scenarioSlugs(it).indexOf(at)
  return i < 0 ? undefined : { scenario: all[i], index: i }
}

/** The boxes Apps and services draws: the containers, the datastores, the outside and the people. */
const drawn = (n: Node) => ['application', 'datastore', 'queue', 'proxy', 'external', 'actor', 'system'].includes(n.kind)

// The engine's two rules over the facts (internal/render: storeOf, queueOf), not a second set: the
// tables a container's code declares live in the one relational store it talks to; the jobs it queues
// go through its one queue store. Two, or none, and nothing is lit rather than guess.
const relational = (tech?: string) => ['postgres', 'mysql', 'mariadb', 'sql', 'cockroach', 'oracle'].some((s) => (tech ?? '').toLowerCase().includes(s))
const queueTech = (tech?: string) => ['redis', 'valkey', 'keydb', 'dragonfly', 'rabbitmq', 'kafka', 'nats', 'activemq', 'sqs'].some((s) => (tech ?? '').toLowerCase().includes(s))

function onlyStore(model: Model, container: string, is: (n: Node) => boolean): string | undefined {
  const nodes = new Map((model.nodes ?? []).map((n) => [n.id, n]))
  const found = new Set<string>()
  for (const e of model.edges ?? []) {
    const n = e.from === container && e.traffic ? nodes.get(e.to) : undefined
    if (n && is(n)) found.add(n.id)
  }
  return found.size === 1 ? [...found][0] : undefined
}

/** Where a container's tables live: its one relational store. */
export const storeOf = (model: Model, container: string) => onlyStore(model, container, (n) => n.kind === 'datastore' && relational(n.technology))

/** Where a container's jobs are queued: its one queue store. */
export const queueOf = (model: Model, container: string) => onlyStore(model, container, (n) => n.kind === 'queue' || (n.kind === 'datastore' && queueTech(n.technology)))

/**
 * The box on Apps and services a fact sits in: itself when it is one; a component's container; a
 * table's store (and the container that declares it); a job's container and its queue store; an entry
 * point's container; both ends of a connection. A group or the system as a whole sits in none.
 */
export function boxesOf(id: string, model: Model): string[] {
  const nodes = new Map((model.nodes ?? []).map((n) => [n.id, n]))
  const up = (nid: string | undefined, depth = 0): string | undefined => {
    const n = nid ? nodes.get(nid) : undefined
    if (!n || depth > 4) return undefined
    return drawn(n) ? n.id : up(n.parent, depth + 1)
  }
  if (id.includes('>')) {
    const [from, to] = id.split('>', 2)
    return [up(from), up(to)].filter((x): x is string => !!x)
  }
  const table = (t: Node | undefined) => {
    const b = up(t?.id)
    if (!b) return []
    const store = storeOf(model, b)
    return store ? [b, store] : [b]
  }
  if (id.startsWith('table:') && !nodes.has(id)) {
    const name = id.slice('table:'.length)
    return table((model.nodes ?? []).find((n) => n.kind === 'table' && n.name === name))
  }
  if (nodes.get(id)?.kind === 'table') return table(nodes.get(id))
  const e = (model.entries ?? []).find((x: Entry) => x.id === id)
  if (e) {
    const b = up(e.container)
    if (!b) return []
    const queue = e.kind === 'job' ? queueOf(model, b) : undefined
    return queue ? [b, queue] : [b]
  }
  const b = up(id)
  return b ? [b] : []
}

/** The tables a set of entry points' flows reach: what a feature stores and reads. */
export function tablesReached(entries: string[], model: Model): string[] {
  const want = new Set(entries)
  const out: string[] = []
  for (const f of model.flows ?? []) {
    if (!want.has(f.entry)) continue
    for (const p of f.participants) if (p.kind === 'table' && p.element && !out.includes(p.element)) out.push(p.element)
  }
  return out
}

/** Every fact a feature rests on, for lighting it: its parts, its ways in, and the tables they reach. */
export const featureFacts = (f: InterpretedGroup, model: Model) => [...(f.parts ?? []), ...f.members, ...tablesReached(f.members, model)]

/** Every box a set of facts sits in, without repeats, in the order first met. */
export function boxesOfAll(ids: string[], model: Model): string[] {
  const out: string[] = []
  for (const id of ids) for (const b of boxesOf(id, model)) if (!out.includes(b)) out.push(b)
  return out
}

/** Where a citation opens, and what it is called — or nothing, for an ID that names no fact here. */
export function citeTarget(id: string, model: Model, it: Interpretation): { label: string; to: Partial<Route> } | undefined {
  const name = (nid: string) => (model.nodes ?? []).find((n) => n.id === nid)?.name ?? nid.replace(/^[a-z]+:/, '')
  const group = /^(feature|concept|area):(\d+)$/.exec(id)
  if (group) {
    const i = Number(group[2]) - 1
    if (group[1] === 'feature') {
      const f = it.features?.[i]
      return f ? { label: f.name, to: { screen: 'feature', focus: featureSlugs(it)[i] } } : undefined
    }
    if (group[1] === 'concept') {
      const c = it.concepts?.[i]
      return c ? { label: c.name, to: { screen: 'data' } } : undefined
    }
    const a = it.areas?.[i]
    return a && it.areas_of ? { label: a.name, to: { screen: 'architecture', level: `component:${it.areas_of}` } } : undefined
  }
  if (id === 'S') return { label: 'what the repository says it is for', to: { screen: 'report' } }
  if (id.includes('>')) {
    const [from, to] = id.split('>', 2)
    return { label: `${name(from)} → ${name(to)}`, to: { screen: 'architecture', level: 'container', focus: from } }
  }
  const e = (model.entries ?? []).find((x) => x.id === id)
  if (e) {
    const what = e.kind === 'http' ? `${e.method} ${e.path}` : e.path || e.id
    return { label: what, to: { screen: 'entries', focus: e.id } }
  }
  if (id.startsWith('table:')) return { label: `table ${id.slice('table:'.length)}`, to: { screen: 'data' } }
  const n = (model.nodes ?? []).find((x) => x.id === id)
  if (!n) return undefined
  if (n.kind === 'component') return { label: n.name, to: { screen: 'component', focus: n.id } }
  return { label: n.name, to: { screen: 'architecture', level: 'container', focus: n.id } }
}

/** How many of a scenario's steps happen while someone waits, and how many later, in the background. */
export function stepSplit(s: Scenario): { wait: number; later: number } {
  const later = s.steps.filter((x) => x.background).length
  return { wait: s.steps.length - later, later }
}

/** "6 steps · 3 while you wait, 3 in the background", or as many as there are. */
export function stepsLine(s: Scenario): string {
  const { wait, later } = stepSplit(s)
  const n = s.steps.length
  const head = `${n} ${n === 1 ? 'step' : 'steps'}`
  if (!later) return `${head} · all while you wait`
  if (!wait) return `${head} · all in the background`
  return `${head} · ${wait} while you wait, ${later} in the background`
}

/** The boxes a scenario has reached by the step at (0-based), and those the current step crosses. */
export function scenarioReach(s: Scenario, at: number, model: Model): { reached: string[]; current: string[] } {
  const reached: string[] = []
  for (const st of s.steps.slice(0, at + 1)) for (const b of boxesOfAll(st.crosses ?? [], model)) if (!reached.includes(b)) reached.push(b)
  return { reached, current: boxesOfAll(s.steps[at]?.crosses ?? [], model) }
}

/** The feature an entry point serves, and the scenarios that cross a part: the one-part view's groups. */
export function featuresThrough(it: Interpretation, box: string, model: Model): InterpretedGroup[] {
  return (it.features ?? []).filter((f) => boxesOfAll(featureFacts(f, model), model).includes(box))
}

export function scenariosThrough(it: Interpretation, box: string, model: Model): Scenario[] {
  return (it.scenarios ?? []).filter((s) => s.steps.some((st) => boxesOfAll(st.crosses ?? [], model).includes(box)))
}

/** The area each component of the divided container is in, by the explanation's grouping. */
export function areaOf(it: Interpretation): Map<string, string> {
  const out = new Map<string, string>()
  for (const a of it.areas ?? []) for (const c of a.members) out.set(c, a.name)
  return out
}
