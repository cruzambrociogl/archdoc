// A scenario as a UML sequence (app-design-screens §9, "As a sequence"), from the flow it follows:
// the engine's stored steps, each at its line — the calls the code makes, the tables its queries
// name, the jobs it queues — drawn between the parts they cross. Nothing is computed beyond
// following the stored IDs to the parts they sit in (the app's first rule).
//
// - A lifeline per part, in the order the story first meets it, the stores and the outside last:
//   the apps and services a story crosses — or, where those would be two lifelines or fewer, too
//   few to say anything, the components of the app the story runs in: one level down.
// - A message for each call between two lifelines, carrying its step's number and what it does:
//   the method it calls, "reads runs", "queues build". A call within one lifeline is that part's own
//   work; a step made only of those is a note on its lifeline.
// - A request's answer comes back dashed and unnumbered, once the steps while you wait are done.
// - What runs later — a step the story marks so, or every step of a job's flow — is in the
//   background: a grey group, its arrows dashed.

import type { Flow, Model, Node, Scenario } from './api'
import { boxesOfAll, queueOf, storeOf } from './explain'

/** How many of a flow's own steps the explanation was given (internal/semantic, maxSteps). */
export const GIVEN = 60

export interface Lane {
  id: string
  name: string
  kind: 'caller' | 'part' | 'store' | 'outside'
}

export interface Message {
  /** The scenario's step it belongs to, from 0. */
  step: number
  from: string
  to: string
  label: string
  /** A call, the request's answer coming back, or a part's own work on its lifeline. */
  kind: 'call' | 'reply' | 'note'
  later: boolean
}

export interface SequenceModel {
  lanes: Lane[]
  messages: Message[]
  /** The app a story inside one app is drawn within, by name; none at the level of apps and services. */
  inside?: string
  /** Read from the flow's steps, or — no flow stored — from what each step crosses alone. */
  from: 'flow' | 'crosses'
}

/** A scenario step happens later when the story says so, and always when the story is a job's. */
export function later(s: Scenario, i: number, model: Model): boolean {
  const entry = (model.entries ?? []).find((e) => e.id === s.flow)
  return !!s.steps[i]?.background || entry?.kind === 'job'
}

/** The same scenario, its steps marked in the background where they are (see `later`). */
export function settle(s: Scenario, model: Model): Scenario {
  const entry = (model.entries ?? []).find((e) => e.id === s.flow)
  if (entry?.kind !== 'job' || s.steps.every((x) => x.background)) return s
  return { ...s, steps: s.steps.map((x) => ({ ...x, background: true })) }
}

/** A few words of a step's sentence, for an arrow: "The worker finds runs that were skipped" → "The worker finds runs that…". */
export function words(text: string, n = 5): string {
  const first = text.split(/[,.;:—]/)[0].trim()
  const all = first.split(/\s+/)
  return all.length > n ? all.slice(0, n).join(' ') + '…' : first
}

export function buildSequence(s: Scenario, model: Model): SequenceModel {
  const nodes = new Map((model.nodes ?? []).map((n) => [n.id, n]))
  const entries = new Map((model.entries ?? []).map((e) => [e.id, e]))
  const flow = (model.flows ?? []).find((f) => f.entry === s.flow)
  const entry = entries.get(s.flow)
  if (!flow || !entry) return fromCrosses(s, model)

  // The box on Apps and services a node sits in: itself, or the nearest one above it.
  const drawn = (n: Node) => ['application', 'datastore', 'queue', 'proxy', 'external', 'actor', 'system'].includes(n.kind)
  const boxOf = (id: string | undefined, depth = 0): string | undefined => {
    const n = id ? nodes.get(id) : undefined
    if (!n || depth > 4) return undefined
    return drawn(n) ? n.id : boxOf(n.parent, depth + 1)
  }
  const home = boxOf(entry.container) ?? entry.container
  const parts = new Map(flow.participants.map((p) => [p.id, p]))
  const own = flow.steps.slice(0, GIVEN)

  // Where a participant sits, at either level.
  const store = storeOf(model, home)
  const queue = queueOf(model, home)
  const jobEntry = (p: Flow['participants'][number]) => (p.element ? entries.get(p.element) : undefined)
  // Tables whose store archdoc does not know — no connection to one is declared — are still a place
  // the code writes to, apart from the code: a lifeline of their own, named for what it is.
  const tables = (box: string) => `tables:${box}`
  const tableLane = (p: Flow['participants'][number]) => {
    const box = boxOf(p.element)
    if (store) return store
    if (box && nodes.get(box)?.kind !== 'application') return box
    return tables(box ?? home)
  }
  const appLane = (id: string): string => {
    const p = parts.get(id)
    if (!p) return home
    if (p.kind === 'table') return tableLane(p)
    if (p.kind === 'job') return queue ?? (jobEntry(p) ? boxOf(jobEntry(p)!.container) ?? home : home)
    if (p.kind === 'unresolved') return 'unresolved'
    return boxOf(p.component) ?? home
  }
  const innerLane = (id: string): string => {
    const p = parts.get(id)
    if (!p) return entry.component ?? home
    if (p.kind === 'table') return tableLane(p)
    if (p.kind === 'job') {
      const j = jobEntry(p)
      const box = j ? boxOf(j.container) : undefined
      return queue ?? (box && box !== home ? box : j?.component ?? home)
    }
    if (p.kind === 'unresolved') return 'unresolved'
    return p.component ?? home
  }

  // Which steps of the flow each sentence stands for: its own, numbered from 1, then its background.
  const ownOf = (i: number) => {
    const st = s.steps[i]
    const out: number[] = []
    for (let k = Math.max(1, st.from); k <= Math.min(st.to, own.length); k++) out.push(k - 1)
    return out
  }
  // One level down when the apps and services the story crosses are too few to say anything: two
  // lifelines or fewer, its own app and one store, say.
  const atApps = new Set<string>()
  s.steps.forEach((_, i) => {
    for (const k of ownOf(i)) {
      const a = appLane(own[k].from)
      const b = appLane(own[k].to)
      if (a !== b) atApps.add(a).add(b)
    }
  })
  const inside = atApps.size <= 2
  const lane = inside ? innerLane : appLane
  const entryLane = inside ? entry.component ?? home : home

  const messages: Message[] = []
  const met: string[] = []
  const meet = (id: string) => {
    if (!met.includes(id)) met.push(id)
  }
  const what = (stTo: string, call: string) => {
    const p = parts.get(stTo)
    if (p?.kind === 'table') return `${call} ${p.name}`
    if (p?.kind === 'job') return `queues ${p.name}`
    if (p?.kind === 'unresolved') return `${call} (address made at run time)`
    return call
  }

  // Who asks: a person, for a route or a command; the queue, for a job.
  const caller = entry.kind === 'job' ? (queue ? queue : undefined) : 'caller'
  const callerName = entry.kind === 'command' ? 'Someone at a terminal' : (model.nodes ?? []).find((n) => n.kind === 'actor')?.name ?? 'User'
  if (caller) {
    meet(caller)
    meet(entryLane)
    const label = entry.kind === 'http' ? `${entry.method} ${entry.path}` : entry.kind === 'command' ? `runs ${entry.path}` : `hands over ${entry.path}`
    messages.push({ step: 0, from: caller, to: entryLane, label, kind: 'call', later: later(s, 0, model) })
  } else meet(entryLane)

  let lastWait = -1
  s.steps.forEach((st, i) => {
    const bg = later(s, i, model)
    if (!bg) lastWait = i
    const mark = messages.length
    for (const k of ownOf(i)) {
      const step = own[k]
      const a = lane(step.from)
      const b = lane(step.to)
      if (a === b) continue // a part's own work
      meet(a)
      meet(b)
      const label = what(step.to, step.call)
      const prev = messages[messages.length - 1]
      // The same exchange again, or another table the same way, is one message: "reads runs, projects".
      if (prev && messages.length > mark && prev.from === a && prev.to === b) {
        if (prev.label === label) continue
        const verb = label.split(' ')[0]
        if (prev.label.startsWith(verb + ' ') && parts.get(step.to)?.kind === 'table') {
          prev.label += `, ${label.slice(verb.length + 1)}`
          continue
        }
      }
      messages.push({ step: i, from: a, to: b, label, kind: 'call', later: bg })
    }
    // What the story queues for later, past the flow's own steps: one job to the next.
    if (st.to > own.length) {
      for (const id of st.crosses ?? []) {
        const j = entries.get(id)
        if (j?.kind !== 'job') continue
        const box = boxOf(j.container) ?? home
        const to = queue ?? (inside && box === home ? j.component ?? box : box)
        const from = messages.length ? messages[messages.length - 1].to : entryLane
        if (from === to) continue
        meet(from)
        meet(to)
        messages.push({ step: i, from, to, label: `queues ${j.path}`, kind: 'call', later: true })
      }
    }
    if (messages.length === mark) {
      // A step made only of a part's own work: a note on its lifeline.
      const at = ownOf(i).length ? lane(own[ownOf(i)[0]].from) : entryLane
      meet(at)
      messages.push({ step: i, from: at, to: at, label: words(st.text, 7), kind: 'note', later: bg })
    }
  })
  if (caller === 'caller' && lastWait >= 0) {
    // The request's answer: back once the steps while you wait are done, before anything later.
    const after = messages.findIndex((m) => m.step > lastWait)
    const reply: Message = { step: lastWait, from: entryLane, to: 'caller', label: 'the answer', kind: 'reply', later: false }
    if (after < 0) messages.push(reply)
    else messages.splice(after, 0, reply)
  }

  const name = (id: string): string => {
    if (id === 'caller') return callerName
    if (id === 'unresolved') return 'an address made at run time'
    if (id.startsWith('tables:')) return `${name(id.slice('tables:'.length))}’s tables`
    const n = nodes.get(id)
    if (n) return n.kind === 'table' ? `table ${n.name}` : n.name
    return id.replace(/^[a-z]+:/, '')
  }
  const kindOf = (id: string): Lane['kind'] => {
    if (id === 'caller') return 'caller'
    if (id === 'unresolved') return 'outside'
    if (id.startsWith('tables:')) return 'store'
    const k = nodes.get(id)?.kind
    if (k === 'datastore' || k === 'queue' || k === 'table') return 'store'
    return k === 'external' ? 'outside' : 'part'
  }
  const rank = (l: Lane) => (l.kind === 'caller' ? 0 : l.kind === 'part' ? 1 : l.kind === 'store' ? 2 : 3)
  const lanes = met.map((id) => ({ id, name: name(id), kind: kindOf(id) })).sort((a, b) => rank(a) - rank(b) || met.indexOf(a.id) - met.indexOf(b.id))
  // A job's queue hands over first, so it stands at the left even though it is a store.
  if (caller && caller !== 'caller') {
    const q = lanes.findIndex((l) => l.id === caller)
    lanes.unshift(...lanes.splice(q, 1))
  }
  return { lanes, messages, inside: inside ? name(home) : undefined, from: 'flow' }
}

/** Without a stored flow: a lifeline per box a step crosses, a message per step, as before. */
function fromCrosses(s: Scenario, model: Model): SequenceModel {
  const nodes = new Map((model.nodes ?? []).map((n) => [n.id, n]))
  const per = s.steps.map((x) => boxesOfAll(x.crosses ?? [], model))
  const met: string[] = []
  for (const boxes of per) for (const b of boxes) if (!met.includes(b)) met.push(b)
  let last = met[0]
  const messages: Message[] = s.steps.map((st, i) => {
    const boxes = per[i]
    const from = boxes[0] ?? last
    const to = boxes.find((b) => b !== from) ?? from
    last = from
    return { step: i, from, to, label: words(st.text), kind: from === to ? 'note' : 'call', later: later(s, i, model) }
  })
  const kind = (id: string): Lane['kind'] => {
    const k = nodes.get(id)?.kind
    return k === 'datastore' || k === 'queue' ? 'store' : k === 'external' ? 'outside' : 'part'
  }
  return { lanes: met.map((id) => ({ id, name: nodes.get(id)?.name ?? id, kind: kind(id) })), messages, from: 'crosses' }
}

/** The arrows of Played on the parts: each the scenario's steps have used, with their numbers; the current one with its words. */
export function playedArrows(s: Scenario, at: number, model: Model, edges: { from: string; to: string }[]): Map<string, { label: string; now: boolean }> {
  const key = (a: string, b: string) => {
    if (edges.some((e) => e.from === a && e.to === b)) return `${a}>${b}`
    if (edges.some((e) => e.from === b && e.to === a)) return `${b}>${a}`
    return undefined
  }
  const used = new Map<string, number[]>()
  const now = new Set<string>()
  s.steps.slice(0, at + 1).forEach((st, i) => {
    const boxes = boxesOfAll(st.crosses ?? [], model)
    for (const b of boxes.slice(1)) {
      const k = key(boxes[0], b)
      if (!k) continue
      used.set(k, [...(used.get(k) ?? []), i + 1].filter((n, j, all) => all.indexOf(n) === j))
      if (i === at) now.add(k)
    }
  })
  const out = new Map<string, { label: string; now: boolean }>()
  for (const [k, ns] of used) {
    out.set(k, now.has(k) ? { label: `${at + 1} · ${words(s.steps[at].text, 4)}`, now: true } : { label: ns.join(', '), now: false })
  }
  return out
}
