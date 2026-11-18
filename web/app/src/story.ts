// One version, told (platform-surface §6.2): its changes grouped by what they mean, in a fixed
// order, and one sentence from the same groups — written by a template from the engine's diff, so
// proven. A repository's What changed tells it at length; a system's, in its sentence.

import type { DiffResponse, Edge, ModelResponse, Node, Provenance } from './api'

/** Where a change landed: an element it touched, and what happened to it, in a word or two ("New route", "Now Valkey 9"). */
export type Landing = { id: string; tag: string }
export type Item = { key: string; text: string; cite?: Provenance; on?: Landing[] }
export type Group = { tag: string; title: string; items: Item[] }

/** The version's changes by meaning, the clauses of its sentence, and whether the architecture changed at all. */
export function tell(d: DiffResponse, before: ModelResponse, after: ModelResponse): { groups: Group[]; clauses: string[]; structural: boolean } {
  const names = new Map<string, Node>()
  for (const m of [before, after]) for (const n of m.model.nodes ?? []) names.set(n.id, n)
  const name = (id: string) => names.get(id)?.name ?? id
  const kind = (id: string) => names.get(id)?.kind ?? ''
  const x = d.diff
  const added = x.added_nodes ?? []
  const removed = x.removed_nodes ?? []
  const addedEdges = x.added_edges ?? []
  const removedEdges = x.removed_edges ?? []
  const changed = x.changed ?? []
  const moved = x.moved ?? []
  const movedTo = new Set(moved.map((m) => m.to))
  const isData = (k: string) => k === 'table' || k === 'datastore' || k === 'queue'
  const wording = changed.filter((c) => c.field === 'description' || c.field === 'label')
  const columns = changed.filter((c) => c.field.startsWith('column '))
  const values = changed.filter((c) => !wording.includes(c) && !columns.includes(c) && !(c.field === 'name' && movedTo.has(c.element)))
  const between = (e: Edge) => !added.some((n) => n.id === e.from || n.id === e.to) && !removed.some((n) => n.id === e.from || n.id === e.to)
  const rewired = [...addedEdges.filter(between), ...removedEdges.filter(between)]
  // A route by its method; a page, a job or a command by what it is.
  const entryText = (e: { kind: string; method: string; path: string; summary?: string }) => `${['page', 'job', 'command'].includes(e.kind) ? e.kind : e.method} ${e.path}${e.summary ? ` — ${e.summary}` : ''}`
  const entryWord = (kind: string) => (kind === 'http' ? 'route' : kind)
  const cap = (t: string) => t.charAt(0).toUpperCase() + t.slice(1)
  // An element by where it is drawn: a component or a table on the part it is inside.
  const home = (id: string) => names.get(id)?.parent ?? id

  // Grouped by what it means, in a fixed order, so the order never shifts (App Findings, round 5).
  const groups: Group[] = [
    {
      tag: '+',
      title: 'New capability',
      items: [
        ...(x.added_entries ?? []).map((e) => ({ key: e.id, text: entryText(e), cite: e.provenance, on: [{ id: e.container, tag: `New ${entryWord(e.kind)}` }] })),
        ...added
          .filter((n) => n.kind === 'component' || n.kind === 'application')
          .map((n) => ({ key: n.id, text: `${n.name}${n.parent ? `, in ${name(n.parent)}` : ''}`, cite: n.provenance, on: [n.parent ? { id: n.parent, tag: 'New part inside' } : { id: n.id, tag: 'New' }] })),
      ],
    },
    { tag: '+', title: 'New outside service', items: added.filter((n) => n.kind === 'external').map((n) => ({ key: n.id, text: n.name, cite: n.provenance, on: [{ id: n.id, tag: 'New' }] })) },
    {
      tag: '±',
      title: 'Data changes',
      items: [
        ...added.filter((n) => isData(n.kind)).map((n) => ({ key: n.id, text: `${n.name} — new`, cite: n.provenance, on: [n.parent ? { id: n.parent, tag: `New ${n.kind}` } : { id: n.id, tag: 'New' }] })),
        ...removed.filter((n) => isData(n.kind)).map((n) => ({ key: `r${n.id}`, text: `${n.name} — gone`, cite: n.provenance, on: [{ id: n.parent ?? n.id, tag: `${cap(n.kind)} gone` }] })),
        ...columns.map((c, i) => ({ key: `c${i}`, text: `${name(c.element)}.${c.field.slice(7)}: ${c.before || '—'} → ${c.after || '—'}`, on: [{ id: home(c.element), tag: 'Columns changed' }] })),
      ],
    },
    {
      tag: '~',
      title: 'Changed',
      items: [
        ...moved.map((m) => ({
          key: m.to,
          text: m.class === 'renamed' ? `${m.was} is now ${m.is}` : `${m.is} ${m.to.startsWith('ext:') ? 'is no longer declared here' : 'is now declared here'}`,
          on: [{ id: m.to, tag: m.class === 'renamed' ? 'Renamed' : 'Moved' }],
        })),
        ...values.map((c, i) => ({
          key: `v${i}`,
          text: `${name(c.element)}: ${c.field} ${c.before || '—'} → ${c.after || '—'}`,
          on: [{ id: home(c.element), tag: c.field === 'technology' && c.after ? `Now ${c.after}` : `${cap(c.field)} changed` }],
        })),
        ...rewired.map((e, i) => ({
          key: `w${i}`,
          text: `${name(e.from)} ${addedEdges.includes(e) ? 'now uses' : 'no longer uses'} ${name(e.to)}`,
          cite: (e.provenance ?? [])[0],
          on: [{ id: e.from, tag: addedEdges.includes(e) ? 'New connection' : 'Connection gone' }, { id: e.to, tag: '' }],
        })),
      ],
    },
    {
      tag: '−',
      title: 'Removed',
      items: [
        ...removed.filter((n) => !isData(n.kind)).map((n) => ({ key: n.id, text: n.name, cite: n.provenance, on: [{ id: n.parent ?? n.id, tag: n.parent ? 'Part gone inside' : 'Gone' }] })),
        ...(x.removed_entries ?? []).map((e) => ({ key: e.id, text: entryText(e), cite: e.provenance, on: [{ id: e.container, tag: `${cap(entryWord(e.kind))} gone` }] })),
      ],
    },
    { tag: '~', title: 'Wording only — doesn’t count', items: wording.map((c, i) => ({ key: `t${i}`, text: `${name(c.element)}: the ${c.field} was rewritten` })) },
  ]
  const full = groups.filter((g) => g.items.length)

  // The story's one sentence, from the same groups: what was added, changed and removed.
  const clauses: string[] = []
  const n = (count: number, one: string, many: string) => `${count} ${count === 1 ? one : many}`
  const ways = (x.added_entries ?? []).length
  const parts = added.filter((p) => p.kind === 'component').length
  const apps = added.filter((p) => p.kind === 'application')
  const outside = added.filter((p) => p.kind === 'external')
  const addedBits = [
    ...apps.map((a) => a.name),
    ...(parts ? [n(parts, 'part', 'parts')] : []),
    ...(ways ? [n(ways, 'way in', 'ways in')] : []),
    ...outside.map((o) => `a call to ${o.name}`),
    ...(added.filter((p) => p.kind === 'table').length ? [n(added.filter((p) => p.kind === 'table').length, 'table', 'tables')] : []),
    ...added.filter((p) => p.kind === 'datastore' || p.kind === 'queue').map((p) => p.name),
  ]
  if (addedBits.length) clauses.push(`added ${join(addedBits)}`)
  const changedBits = [...(columns.length ? [n(columns.length, 'column', 'columns')] : []), ...(moved.length ? [n(moved.length, 'name', 'names')] : []), ...(rewired.length ? [n(rewired.length, 'connection', 'connections')] : []), ...(values.length ? [n(values.length, 'detail', 'details')] : [])]
  if (changedBits.length) clauses.push(`changed ${join(changedBits)}`)
  const gone = removed.length + (x.removed_entries ?? []).length
  if (gone) clauses.push(`removed ${n(gone, 'thing', 'things')}`)
  const structural = full.some((g) => !g.title.startsWith('Wording'))

  return { groups, clauses, structural }
}

export function join(xs: string[]): string {
  return xs.length <= 1 ? (xs[0] ?? '') : `${xs.slice(0, -1).join(', ')} and ${xs[xs.length - 1]}`
}

/**
 * Where a change landed, on a diagram whose boxes are `drawn`: each element it touched, lifted to the
 * box it is drawn inside — a component to its application — with what happened to it. Two things
 * on one box say both. What it touched that is drawn nowhere here, a part gone, is left out.
 */
export function landed(on: Landing[], parentOf: (id: string) => string | undefined, drawn: Set<string>): { lit: Set<string>; tags: Map<string, string> } {
  const lit = new Set<string>()
  const tags = new Map<string, string>()
  for (const l of on) {
    let id: string | undefined = l.id
    for (let hops = 0; id && !drawn.has(id) && hops < 8; hops++) id = parentOf(id)
    if (!id || !drawn.has(id)) continue
    lit.add(id)
    if (!l.tag) continue
    const had = tags.get(id)
    if (!had) tags.set(id, l.tag)
    else if (!had.split(' · ').includes(l.tag)) tags.set(id, `${had} · ${l.tag}`)
  }
  return { lit, tags }
}
