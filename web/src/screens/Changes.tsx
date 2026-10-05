import type { ReactNode } from 'react'
import type { DiffResponse, Edge, ModelResponse, Node, Version } from '../api'
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
  for (const m of [before.data, after.data]) for (const n of m?.model.nodes ?? []) names.set(n.id, n.name)
  const name = (id: string) => names.get(id) ?? id
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
              <button className="text-link compare-show" onClick={() => props.go({ screen: 'explorer', v: props.route.v })}>
                Show on diagram
              </button>
            </div>
            {diff.loading && <Loading />}
            {diff.error && <Failure error={diff.error} />}
            {diff.data && <Diff d={diff.data} name={name} />}
          </>
        )}
      </div>
    </div>
  )
}

function Diff({ d, name }: { d: DiffResponse; name: (id: string) => string }) {
  const x = d.diff
  const added = x.added_nodes ?? []
  const removed = x.removed_nodes ?? []
  const addedE = x.added_edges ?? []
  const removedE = x.removed_edges ?? []
  const changed = x.changed ?? []
  const structuralCount = added.length + removed.length + addedE.length + removedE.length

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
            +{added.length + addedE.length} appeared · −{removed.length + removedE.length} disappeared
          </div>
        </div>
        <div>
          <Eyebrow>Values</Eyebrow>
          <div className="change-counts">{changed.length} changed</div>
        </div>
      </div>
      {structuralCount === 0 && changed.length > 0 && (
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
