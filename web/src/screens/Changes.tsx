import type { ReactNode } from 'react'
import type { DiffResponse, Edge, Entry, ModelResponse, Node, Provenance, Version } from '../api'
import { useApi } from '../api'
import type { Route } from '../route'
import { Cite } from '../ui/Cite'
import { Eyebrow, Failure, Loading } from '../ui/marks'

/**
 * The change lens (surface-spec §5.12): what changed in the architecture between two versions.
 * A version exists only when the architecture changed, so every entry in the rail is a real
 * change. Structural change — something appeared, disappeared or was rewired — is kept apart
 * from changed values, so a relabel never reads as a rewiring.
 */
export function Changes(props: { versions: Version[]; route: Route; go: (r: Partial<Route>, o?: { keep?: boolean }) => void }) {
  const sorted = [...props.versions].sort((a, b) => b.id - a.id)
  const to = props.route.v ?? sorted[0]?.id
  const base = props.route.from ?? sorted.find((v) => to !== undefined && v.id < to)?.id
  const ready = !!(to && base && base !== to)
  const diff = useApi<DiffResponse>(ready ? `/api/diff?from=${base}&to=${to}` : null)
  // Both versions' models, so relationships and changed values read by name, not by id.
  const before = useApi<ModelResponse>(ready ? `/api/model?version=${base}` : null)
  const after = useApi<ModelResponse>(ready ? `/api/model?version=${to}` : null)
  const names = new Map<string, string>()
  const kinds = new Map<string, string>()
  for (const m of [before.data, after.data])
    for (const n of m?.model.nodes ?? []) {
      names.set(n.id, n.name)
      kinds.set(n.id, n.kind)
    }
  const name = (id: string) => names.get(id) ?? id
  const kind = (id: string) => kinds.get(id) ?? ''
  const label = (id?: number) => {
    const v = sorted.find((x) => x.id === id)
    return v ? `v${v.id} · ${new Date(v.created_at).toLocaleDateString(undefined, { day: 'numeric', month: 'short' })}${v.commit ? ` · ${v.commit.slice(0, 7)}` : ''}` : ''
  }

  return (
    <div className="changes">
      <aside className="version-rail">
        <div className="eyebrow rail-head">Versions</div>
        {sorted.map((v) => (
          <button
            key={v.id}
            className={`rail-item ${v.id === to ? 'on' : ''} ${v.id === base ? 'base' : ''}`}
            onClick={() => props.go({ screen: 'changes', v: v.id === sorted[0]?.id ? undefined : v.id })}
          >
            <span className="rail-line">
              <span className="mono">v{v.id}</span>
              <span className="mono muted">{v.commit ? v.commit.slice(0, 7) : '—'}</span>
            </span>
            <span className="rail-sub">
              {new Date(v.created_at).toLocaleString(undefined, { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' })}
              {v.id === base ? ' · compared with' : ''}
            </span>
          </button>
        ))}
      </aside>

      <div className="changes-main reading">
        <Eyebrow>Changes · the change lens</Eyebrow>
        {!to || sorted.length < 2 || !base ? (
          <>
            <h1 className="page-title small">{to ? `v${to}` : 'No versions'}</h1>
            <p className="lede">This is the first version recorded. Changes appear here once the architecture changes and archdoc generate records another.</p>
          </>
        ) : (
          <>
            <h1 className="page-title small">
              v{base} → v{to}
            </h1>
            <div className="compare-row">
              <label className="compare-pick">
                <select value={base} onChange={(e) => props.go({ screen: 'changes', v: props.route.v, from: Number(e.target.value) })}>
                  {sorted
                    .filter((v) => v.id !== to)
                    .map((v) => (
                      <option key={v.id} value={v.id}>
                        {label(v.id)}
                      </option>
                    ))}
                </select>
              </label>
              <span className="muted">→</span>
              <span className="compare-fixed mono">{label(to)}</span>
              <button className="text-link compare-show" onClick={() => props.go({ screen: 'explorer', v: props.route.v, from: base })}>
                Show on diagram
              </button>
            </div>
            {diff.loading && <Loading />}
            {diff.error && <Failure error={diff.error} />}
            {diff.data && <Diff d={diff.data} name={name} kind={kind} />}
          </>
        )}
      </div>
    </div>
  )
}

function Diff({ d, name, kind }: { d: DiffResponse; name: (id: string) => string; kind: (id: string) => string }) {
  const x = d.diff
  // What configuration describes gets a card each; what the code declares inside the containers —
  // components, tables, routes — comes by the dozen, and is grouped, its list folded.
  const part = (k: string) => k === 'component' || k === 'table'
  const allAdded = x.added_nodes ?? []
  const allRemoved = x.removed_nodes ?? []
  const added = allAdded.filter((n) => !part(n.kind))
  const removed = allRemoved.filter((n) => !part(n.kind))
  const inside = (e: Edge) => part(kind(e.from)) || part(kind(e.to))
  const addedE = (x.added_edges ?? []).filter((e) => !inside(e))
  const removedE = (x.removed_edges ?? []).filter((e) => !inside(e))
  const addedInside = (x.added_edges ?? []).filter(inside)
  const removedInside = (x.removed_edges ?? []).filter(inside)
  const allChanged = x.changed ?? []
  const changed = allChanged.filter((c) => !c.field.startsWith('column '))
  const columns = allChanged.filter((c) => c.field.startsWith('column '))
  const addedEntries = x.added_entries ?? []
  const removedEntries = x.removed_entries ?? []
  const appeared = allAdded.length + (x.added_edges ?? []).length + addedEntries.length
  const disappeared = allRemoved.length + (x.removed_edges ?? []).length + removedEntries.length
  const structuralCount = appeared + disappeared
  const groups: { glyph: string; title: string; rows: { key: string; text: string; cite?: Provenance }[] }[] = []
  for (const [k, label] of [
    ['component', 'components'],
    ['table', 'tables'],
  ] as const) {
    const a = allAdded.filter((n) => n.kind === k)
    const r = allRemoved.filter((n) => n.kind === k)
    if (a.length) groups.push({ glyph: '+', title: `${a.length} ${label} appeared`, rows: a.map((n) => ({ key: n.id, text: `${n.name} · in ${name(n.parent ?? '')}`, cite: n.provenance })) })
    if (r.length) groups.push({ glyph: '−', title: `${r.length} ${label} disappeared`, rows: r.map((n) => ({ key: n.id, text: `${n.name} · in ${name(n.parent ?? '')}`, cite: n.provenance })) })
  }
  const entryText = (e: Entry) => `${e.kind === 'page' ? 'page' : e.method} ${e.path}${e.summary ? ` — ${e.summary}` : ''}`
  if (addedEntries.length) groups.push({ glyph: '+', title: `${addedEntries.length} routes and pages appeared`, rows: addedEntries.map((e) => ({ key: e.id, text: entryText(e), cite: e.provenance })) })
  if (removedEntries.length) groups.push({ glyph: '−', title: `${removedEntries.length} routes and pages disappeared`, rows: removedEntries.map((e) => ({ key: e.id, text: entryText(e), cite: e.provenance })) })
  const insideText = (e: Edge) => `${name(e.from)} → ${name(e.to)}`
  if (addedInside.length) groups.push({ glyph: '+', title: `${addedInside.length} uses and foreign keys appeared`, rows: addedInside.map((e) => ({ key: `${e.from}>${e.to}`, text: insideText(e), cite: (e.provenance ?? [])[0] })) })
  if (removedInside.length) groups.push({ glyph: '−', title: `${removedInside.length} uses and foreign keys disappeared`, rows: removedInside.map((e) => ({ key: `${e.from}>${e.to}`, text: insideText(e), cite: (e.provenance ?? [])[0] })) })
  if (columns.length) groups.push({ glyph: '±', title: `${columns.length} columns changed`, rows: columns.map((c, i) => ({ key: `${c.element}${c.field}${i}`, text: `${name(c.element)}.${c.field.slice(7)}: ${c.before || '—'} → ${c.after || '—'}` })) })

  if (d.empty) {
    return (
      <p className="lede-quiet">
        No difference between v{d.from} and v{d.to}. The version was recorded because something archdoc stores changed, not
        the architecture.
      </p>
    )
  }

  const edge = (e: Edge) => `${name(e.from)} → ${name(e.to)}${e.label ? ` · ${e.label}` : ''}`

  return (
    <>
      <div className="change-summary">
        <div>
          <Eyebrow>Structural</Eyebrow>
          <div className="change-counts">
            +{appeared} appeared · −{disappeared} disappeared
          </div>
        </div>
        <div>
          <Eyebrow>Values</Eyebrow>
          <div className="change-counts">{allChanged.length} changed</div>
        </div>
      </div>
      {structuralCount === 0 && allChanged.length > 0 && (
        <p className="lede-quiet">Only values changed — nothing appeared, disappeared or was rewired.</p>
      )}

      {added.map((n) => (
        <Card key={`a${n.id}`} glyph="+" title={`${n.name} appeared`} kind="element">
          <Side label={`before · v${d.from}`}>
            <span className="muted">—</span>
          </Side>
          <Side label={`after · v${d.to}`}>
            <NodeLine n={n} />
          </Side>
        </Card>
      ))}
      {removed.map((n) => (
        <Card key={`r${n.id}`} glyph="−" title={`${n.name} disappeared`} kind="element">
          <Side label={`before · v${d.from}`}>
            <NodeLine n={n} />
          </Side>
          <Side label={`after · v${d.to}`}>
            <span className="muted">—</span>
          </Side>
        </Card>
      ))}
      {addedE.map((e, i) => (
        <Card key={`ae${i}`} glyph="+" title={`${name(e.from)} → ${name(e.to)}`} kind="relationship">
          <Side label={`before · v${d.from}`}>
            <span className="muted">—</span>
          </Side>
          <Side label={`after · v${d.to}`}>
            <span>{edge(e)}</span>
            <Cites e={e} />
          </Side>
        </Card>
      ))}
      {removedE.map((e, i) => (
        <Card key={`re${i}`} glyph="−" title={`${name(e.from)} → ${name(e.to)}`} kind="relationship">
          <Side label={`before · v${d.from}`}>
            <span>{edge(e)}</span>
            <Cites e={e} />
          </Side>
          <Side label={`after · v${d.to}`}>
            <span className="muted">—</span>
          </Side>
        </Card>
      ))}
      {changed.map((c, i) => (
        <Card key={`c${i}`} glyph="±" hollow title={`${name(c.element)} · ${c.field}`} kind="value">
          <Side label={`before · v${d.from}`}>{c.before || <span className="muted">—</span>}</Side>
          <Side label={`after · v${d.to}`}>{c.after || <span className="muted">—</span>}</Side>
        </Card>
      ))}
      {groups.length > 0 && <div className="eyebrow change-inside">Inside the containers · from the code</div>}
      {groups.map((g) => (
        <details key={g.title} className="change-card change-group">
          <summary className="change-card-head">
            <span className={`delta-tag ${g.glyph === '±' ? 'hollow' : ''}`}>{g.glyph}</span>
            <span className="change-title">{g.title}</span>
            <span className="eyebrow change-kind">show</span>
          </summary>
          <ul className="change-rows">
            {g.rows.map((r) => (
              <li key={r.key}>
                <span className="mono small">{r.text}</span>
                {r.cite && <Cite p={r.cite} compact />}
              </li>
            ))}
          </ul>
        </details>
      ))}
    </>
  )
}

function NodeLine({ n }: { n: Node }) {
  return (
    <>
      <span>
        {n.name}
        {n.technology ? <span className="muted"> · {n.technology}</span> : null}
      </span>
      <div className="side-cite">
        <Cite p={n.provenance} />
      </div>
    </>
  )
}

function Cites({ e }: { e: Edge }) {
  return (
    <div className="side-cite">
      {(e.provenance ?? []).map((p, i) => (
        <Cite key={i} p={p} />
      ))}
    </div>
  )
}

function Card(props: { glyph: string; title: string; kind: string; hollow?: boolean; children: ReactNode }) {
  return (
    <div className="change-card">
      <div className="change-card-head">
        <span className={`delta-tag ${props.hollow ? 'hollow' : ''}`}>{props.glyph}</span>
        <span className="change-title">{props.title}</span>
        <span className="eyebrow change-kind">{props.kind}</span>
      </div>
      <div className="change-sides">{props.children}</div>
    </div>
  )
}

function Side({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="change-side">
      <div className="side-label">{label}</div>
      <div className="side-body">{children}</div>
    </div>
  )
}
