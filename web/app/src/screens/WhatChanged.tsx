import { useEffect, useMemo, useState } from 'react'
import type { DiffResponse, Edge, ModelResponse, Node, Provenance, Session, Version } from '../api'
import { useApi } from '../api'
import { useCapabilities } from '../source'
import type { Route } from '../route'
import { Cite } from '../ui/Cite'
import { Failure, Loading } from '../ui/marks'
import { Changes } from './Changes'

/**
 * What changed (platform-surface §6.2): what just changed in the architecture? A strip of the
 * versions — which is also the picker between them — then one version told as a story, its changes
 * grouped by what they mean, each with its citation in view. Without AI the story is written by a
 * template from the engine's diff, so it is proven. The detailed comparison of any two versions is
 * still here, under Advanced, intact.
 */
export function WhatChanged(props: { versions: Version[]; route: Route; go: (r: Partial<Route>, o?: { keep?: boolean }) => void; root: string; session?: Session }) {
  const can = useCapabilities()
  const [advanced, setAdvanced] = useState(props.route.from !== undefined)
  const sorted = useMemo(() => [...props.versions].sort((a, b) => b.id - a.id), [props.versions])
  const to = props.route.v ?? sorted[0]?.id
  const base = sorted.find((v) => to !== undefined && v.id < to)?.id
  const ready = !!(to && base)
  const diff = useApi<DiffResponse>(ready ? `/api/diff?from=${base}&to=${to}` : null)
  const before = useApi<ModelResponse>(ready ? `/api/model?version=${base}` : null)
  const after = useApi<ModelResponse>(to ? `/api/model?version=${to}` : null)
  const seen = useSeen(props.root, sorted[0]?.id)

  const newer = seen === undefined ? 0 : sorted.filter((v) => v.id > seen).length
  const since =
    can.place === 'published'
      ? 'since the version it was published against'
      : seen === undefined
        ? ''
        : newer
          ? `${newer} ${newer === 1 ? 'version changed' : 'versions changed'} the architecture since you last opened it here`
          : 'Nothing has changed since you last opened it here'

  return (
    <div className="reading what-changed">
      <div className="page-q">What just changed?</div>

      {sorted.length > 1 && (
        <div className="version-strip" role="group" aria-label="Versions">
          {since && <div className="version-strip-since">{since}</div>}
          <div className="version-strip-row">
            {sorted.slice(0, 8).map((v) => (
              <button
                key={v.id}
                className={`version-chip ${v.id === to ? 'on' : ''} ${seen !== undefined && v.id > seen ? 'new' : ''}`}
                aria-pressed={v.id === to}
                onClick={() => props.go({ screen: 'changes', v: v.id === sorted[0].id ? undefined : v.id })}
              >
                <span className="version-chip-day">{day(v.created_at)}</span>
                <span className="mono version-chip-commit">{v.commit ? v.commit.slice(0, 7) : `v${v.id}`}</span>
              </button>
            ))}
          </div>
        </div>
      )}

      {!ready ? (
        <p className="story-answer">
          This is the first version archdoc recorded{to ? ` (${label(sorted.find((v) => v.id === to))})` : ''}: there is nothing before it to compare with. The next version that changes the architecture
          will be told here.
        </p>
      ) : diff.error ? (
        <Failure error={diff.error} />
      ) : !diff.data || !after.data || !before.data ? (
        <Loading />
      ) : (
        <Story d={diff.data} before={before.data} after={after.data} label={label(sorted.find((v) => v.id === to))} go={props.go} base={base!} to={to!} />
      )}

      {sorted.length > 1 && (
        <div className="advanced">
          <button className="evidence-toggle" aria-expanded={advanced} onClick={() => setAdvanced((a) => !a)}>
            <span className="caret">{advanced ? '▾' : '▸'}</span> Advanced <span className="muted">every version, and comparing any two</span>
          </button>
          {advanced && (
            <div className="advanced-body">
              <Changes versions={props.versions} route={props.route} go={props.go} />
            </div>
          )}
        </div>
      )}
    </div>
  )
}

type Item = { key: string; text: string; cite?: Provenance }
type Group = { tag: string; title: string; items: Item[] }

/** One version, told: a sentence, then its changes by what they mean (§6.2). */
function Story(props: { d: DiffResponse; before: ModelResponse; after: ModelResponse; label: string; base: number; to: number; go: (r: Partial<Route>) => void }) {
  const { d } = props
  const names = new Map<string, Node>()
  for (const m of [props.before, props.after]) for (const n of m.model.nodes ?? []) names.set(n.id, n)
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

  // Grouped by what it means, in a fixed order, so the order never shifts (App Findings, round 5).
  const groups: Group[] = [
    {
      tag: '+',
      title: 'New capability',
      items: [
        ...(x.added_entries ?? []).map((e) => ({ key: e.id, text: entryText(e), cite: e.provenance })),
        ...added.filter((n) => n.kind === 'component' || n.kind === 'application').map((n) => ({ key: n.id, text: `${n.name}${n.parent ? `, in ${name(n.parent)}` : ''}`, cite: n.provenance })),
      ],
    },
    { tag: '+', title: 'New outside service', items: added.filter((n) => n.kind === 'external').map((n) => ({ key: n.id, text: n.name, cite: n.provenance })) },
    {
      tag: '±',
      title: 'Data changes',
      items: [
        ...added.filter((n) => isData(n.kind)).map((n) => ({ key: n.id, text: `${n.name} — new`, cite: n.provenance })),
        ...removed.filter((n) => isData(n.kind)).map((n) => ({ key: `r${n.id}`, text: `${n.name} — gone`, cite: n.provenance })),
        ...columns.map((c, i) => ({ key: `c${i}`, text: `${name(c.element)}.${c.field.slice(7)}: ${c.before || '—'} → ${c.after || '—'}` })),
      ],
    },
    {
      tag: '~',
      title: 'Changed',
      items: [
        ...moved.map((m) => ({ key: m.to, text: m.class === 'renamed' ? `${m.was} is now ${m.is}` : `${m.is} ${m.to.startsWith('ext:') ? 'is no longer declared here' : 'is now declared here'}` })),
        ...values.map((c, i) => ({ key: `v${i}`, text: `${name(c.element)}: ${c.field} ${c.before || '—'} → ${c.after || '—'}` })),
        ...rewired.map((e, i) => ({ key: `w${i}`, text: `${name(e.from)} ${addedEdges.includes(e) ? 'now uses' : 'no longer uses'} ${name(e.to)}`, cite: (e.provenance ?? [])[0] })),
      ],
    },
    {
      tag: '−',
      title: 'Removed',
      items: [
        ...removed.filter((n) => !isData(n.kind)).map((n) => ({ key: n.id, text: n.name, cite: n.provenance })),
        ...(x.removed_entries ?? []).map((e) => ({ key: e.id, text: entryText(e), cite: e.provenance })),
      ],
    },
    { tag: '~', title: 'Wording only — doesn’t count', items: wording.map((c, i) => ({ key: `t${i}`, text: `${name(c.element)}: the ${c.field} was rewritten` })) },
  ]
  const full = groups.filter((g) => g.items.length)
  const empty = groups.filter((g) => !g.items.length && !g.title.startsWith('Wording'))

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

  return (
    <>
      <p className="story-answer">
        {structural ? `This version (${props.label}) ${clauses.join('; ')}.` : `This version (${props.label}) changed code, but not the architecture.`}
      </p>
      <div className="overview-source">
        <span>Written from the engine's comparison with the version before — proven, not interpreted.</span>
        {structural && (
          <button className="link-btn" onClick={() => props.go({ screen: 'explorer', level: 'container', v: props.to === undefined ? undefined : props.to, from: props.base })}>
            See it on the map →
          </button>
        )}
      </div>
      <div className="meanings">
        {full.map((g) => (
          <section key={g.title} className={`meaning-group ${g.title.startsWith('Wording') ? 'wording' : ''}`}>
            <h2 className="meaning-head">
              <span className="meaning-tag">{g.tag}</span>
              {g.title}
              <span className="muted"> · {g.items.length}</span>
            </h2>
            <ul>
              {g.items.slice(0, 5).map((it) => (
                <li key={it.key}>
                  <span className="meaning-text">{it.text}</span>
                  {it.cite?.file && <Cite p={it.cite} compact />}
                </li>
              ))}
            </ul>
            {g.items.length > 5 && <MoreItems items={g.items.slice(5)} />}
          </section>
        ))}
        {empty.length > 0 && <p className="meaning-nothing">{nothingLine(empty.map((g) => g.title))}</p>}
      </div>
    </>
  )
}

/** Lists stop at five (rule 15); the rest is one deliberate step away. */
function MoreItems({ items }: { items: Item[] }) {
  const [open, setOpen] = useState(false)
  if (!open)
    return (
      <button className="link-btn meaning-more" onClick={() => setOpen(true)}>
        {items.length} more →
      </button>
    )
  return (
    <ul>
      {items.map((it) => (
        <li key={it.key}>
          <span className="meaning-text">{it.text}</span>
          {it.cite?.file && <Cite p={it.cite} compact />}
        </li>
      ))}
    </ul>
  )
}

/** The empty groups, in one closing line (App Findings, round 5): "No new outside service; nothing removed." */
function nothingLine(titles: string[]): string {
  const phrase: Record<string, string> = {
    'New capability': 'no new capability',
    'New outside service': 'no new outside service',
    'Data changes': 'no data change',
    Changed: 'nothing changed',
    Removed: 'nothing removed',
  }
  const s = join(titles.map((t) => phrase[t] ?? t.toLowerCase()))
  return s.charAt(0).toUpperCase() + s.slice(1) + '.'
}

function join(xs: string[]): string {
  return xs.length <= 1 ? (xs[0] ?? '') : `${xs.slice(0, -1).join(', ')} and ${xs[xs.length - 1]}`
}

const day = (iso: string) => new Date(iso).toLocaleDateString(undefined, { day: 'numeric', month: 'short' })
const label = (v?: Version) => (v ? `version ${v.id}${v.commit ? ` · ${v.commit.slice(0, 7)}` : ''} · ${day(v.created_at)}` : '')

/**
 * Which version this browser last showed for this repository (§4.5: locally, "since you last
 * opened it on this machine"). Read once, then moved to the latest. Storage may be blocked; then
 * there is no "since", only the versions.
 */
function useSeen(root: string, latest?: number): number | undefined {
  const key = `archdoc.seen.${root}`
  const [seen] = useState<number | undefined>(() => {
    try {
      const v = localStorage.getItem(key)
      return v === null ? undefined : Number(v)
    } catch {
      return undefined
    }
  })
  useEffect(() => {
    if (latest === undefined) return
    try {
      localStorage.setItem(key, String(latest))
    } catch {
      /* blocked: nothing remembered */
    }
  }, [key, latest])
  return seen
}
