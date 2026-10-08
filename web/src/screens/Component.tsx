import type { ModelResponse } from '../api'
import { componentLevel, dataLevel, useApi } from '../api'
import type { Route } from '../route'
import { Cite } from '../ui/Cite'
import { KindTile } from '../ui/KindTile'
import { Eyebrow, Failure, Loading, TruthMark } from '../ui/marks'

/**
 * A component's page (surface-spec §5.4, F-19): what this part of the system is, where it is in
 * the code, and what it touches — every line of it a stored fact with its citation, except the
 * sentences a model wrote, which are marked and cite the facts they rest on.
 */
export function Component(props: { version: number | null; route: Route; go: (r: Partial<Route>) => void }) {
  const q = props.version ? `?version=${props.version}` : ''
  const m = useApi<ModelResponse>(`/api/model${q}`)
  if (m.error) return <Failure error={m.error} />
  if (!m.data) return <Loading />
  const model = m.data.model
  const nodes = model.nodes ?? []
  const byId = new Map(nodes.map((n) => [n.id, n]))
  const node = byId.get(props.route.focus ?? '')
  if (!node || node.kind !== 'component') {
    return (
      <div className="reading">
        <p className="lede-quiet">No component is named in the address. Open one from the explorer's Components level.</p>
      </div>
    )
  }
  const name = (id: string) => byId.get(id)?.name ?? id
  const container = node.parent ?? ''
  const siblings = nodes.filter((n) => n.kind === 'component' && n.parent === container)
  const at = siblings.findIndex((n) => n.id === node.id)
  const edges = model.edges ?? []
  const uses = edges.filter((e) => e.from === node.id)
  const usedBy = edges.filter((e) => e.to === node.id && byId.get(e.from)?.kind === 'component')
  const entries = (model.entries ?? []).filter((e) => e.component === node.id)
  const files = new Set(node.files ?? [])
  const tables = nodes.filter((n) => n.kind === 'table' && files.has(n.dir ?? ''))
  const deps = (model.dependencies ?? []).filter((d) => (d.components ?? []).includes(node.id))
  const unresolved = (model.unresolved ?? []).filter((u) => u.component === node.id)
  const flows = (model.flows ?? []).filter((f) => f.participants.some((p) => p.component === node.id))
  const explanation = (model.explanations ?? []).find((x) => x.element === node.id)
  const open = (id: string) => props.go({ screen: 'component', focus: id })

  return (
    <div className="reading wide component-page">
      <Eyebrow>
        Component ·{' '}
        <button className="text-link" onClick={() => props.go({ screen: 'explorer', level: componentLevel(container) })}>
          {name(container)}
        </button>
      </Eyebrow>
      <h1 className="page-title">
        <KindTile kind="component" /> {node.name}
      </h1>
      <p className="lede-note">
        <span className="mono">{node.dir}</span> · {(node.files ?? []).length} files · {(node.lines ?? 0).toLocaleString()} lines · {node.technology} ·{' '}
        <Cite p={node.provenance} compact />{' '}
        <button className="text-link" onClick={() => props.go({ screen: 'explorer', level: componentLevel(container), focus: node.id })}>
          Show on the diagram ⤴
        </button>
      </p>

      {explanation ? (
        <section>
          <Eyebrow>What it does · interpreted</Eyebrow>
          {explanation.claims.map((c, i) => (
            <div key={i} className="claim">
              <p className="interpreted lede">
                <TruthMark state="interpreted" /> {c.text}
              </p>
              <div className="relation-cites">
                {c.cites.map((p, j) => (
                  <span key={j} title={c.facts[j]}>
                    <Cite p={p} compact />
                  </span>
                ))}
              </div>
            </div>
          ))}
          <p className="small muted">
            Written by {explanation.provenance.note} from the facts on this page; every sentence cites them.
            {explanation.stale && ' Those facts have changed since: this is the last answer, kept until generate --explain asks again.'}
          </p>
        </section>
      ) : (
        <p className="lede-quiet">
          No description: nothing in the code states one, and no model has been asked. <span className="mono">archdoc generate --explain</span> asks, and every
          sentence it keeps cites the facts below.
        </p>
      )}

      <div className="component-grid">
        <section>
          <Eyebrow>Uses · {uses.length}</Eyebrow>
          {uses.length === 0 && <p className="lede-quiet">Nothing else in this container.</p>}
          {uses.map((e) => (
            <Row key={e.to} onClick={() => open(e.to)} text={name(e.to)} meta={`${e.weight ?? 1} imports`} cite={(e.provenance ?? [])[0]} />
          ))}
        </section>
        <section>
          <Eyebrow>Used by · {usedBy.length}</Eyebrow>
          {usedBy.length === 0 && <p className="lede-quiet">Nothing else in this container.</p>}
          {usedBy.map((e) => (
            <Row key={e.from} onClick={() => open(e.from)} text={name(e.from)} meta={`${e.weight ?? 1} imports`} cite={(e.provenance ?? [])[0]} />
          ))}
        </section>
        {entries.length > 0 && (
          <section>
            <Eyebrow>Ways in · {entries.length}</Eyebrow>
            {entries.slice(0, 40).map((e) => (
              <Row
                key={e.id}
                onClick={() => props.go({ screen: 'features', focus: e.id })}
                text={`${e.kind === 'page' ? '' : e.method + ' '}${e.path}`}
                meta={e.summary}
                cite={e.provenance}
                mono
              />
            ))}
            {entries.length > 40 && (
              <button className="text-link" onClick={() => props.go({ screen: 'features' })}>
                and {entries.length - 40} more on Features
              </button>
            )}
          </section>
        )}
        {tables.length > 0 && (
          <section>
            <Eyebrow>Tables it declares · {tables.length}</Eyebrow>
            {tables.map((t) => (
              <Row
                key={t.id}
                onClick={() => props.go({ screen: 'explorer', level: dataLevel(container), focus: t.id })}
                text={t.name}
                meta={`${(t.columns ?? []).length} columns`}
                cite={t.provenance}
                mono
              />
            ))}
          </section>
        )}
        {deps.length > 0 && (
          <section>
            <Eyebrow>Packages it imports · {deps.length}</Eyebrow>
            {deps.map((d) => (
              <Row key={d.name} text={d.name} meta={d.version} cite={d.provenance} mono />
            ))}
          </section>
        )}
        {(unresolved.length > 0 || flows.length > 0) && (
          <section>
            <Eyebrow>At run time</Eyebrow>
            {flows.length > 0 && <p className="small">Takes part in {flows.length} flows — each drawn on its route, on Features.</p>}
            {unresolved.map((u, i) => (
              <div key={i} className="page-row">
                <span className="small">
                  <TruthMark state="unresolved" /> <span className="mono">{u.what}</span> — an address computed at run time
                </span>
                <Cite p={u.provenance} compact />
              </div>
            ))}
          </section>
        )}
        <section>
          <details className="files">
            <summary>
              <Eyebrow>{(node.files ?? []).length} files</Eyebrow>
            </summary>
            <ul className="file-list">
              {(node.files ?? []).map((f) => (
                <li key={f}>
                  <Cite p={{ file: f, line: 1 }} compact />
                </li>
              ))}
            </ul>
          </details>
        </section>
      </div>

      <div className="component-pager">
        {at > 0 ? (
          <button className="text-link" onClick={() => open(siblings[at - 1].id)}>
            ← {siblings[at - 1].name}
          </button>
        ) : (
          <span />
        )}
        {at < siblings.length - 1 && (
          <button className="text-link" onClick={() => open(siblings[at + 1].id)}>
            {siblings[at + 1].name} →
          </button>
        )}
      </div>
    </div>
  )
}

function Row(props: { text: string; meta?: string; cite?: { file: string; line: number }; onClick?: () => void; mono?: boolean }) {
  return (
    <div className="page-row">
      <span>
        {props.onClick ? (
          <button className={`text-link ${props.mono ? 'mono small' : ''}`} onClick={props.onClick}>
            {props.text}
          </button>
        ) : (
          <span className={props.mono ? 'mono small' : ''}>{props.text}</span>
        )}
        {props.meta && <span className="muted small"> · {props.meta}</span>}
      </span>
      {props.cite && <Cite p={props.cite} compact />}
    </div>
  )
}
