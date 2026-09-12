import { useEffect, useRef, useState } from 'react'
import type { Edge, Model, ModelResponse, Node } from './api'
import { useApi } from './api'
import { Failure, Loading, Prov } from './Prov'

type View = 'container' | 'context'

/**
 * The diagram is the SVG the engine commits to the repository, byte for byte — the app draws
 * nothing itself. Every element in it is a <g id="…">, which is all the page needs to make it
 * clickable.
 */
export function Canvas({ version }: { version: number | null }) {
  const [view, setView] = useState<View>('container')
  const [selected, setSelected] = useState<string | null>(null)
  const [fit, setFit] = useState(true)
  const q = version ? `&version=${version}` : ''
  const svg = useApi<string>(`/api/svg?view=${view}${q}`, 'text')
  const model = useApi<ModelResponse>(`/api/model?${q.slice(1)}`)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    ref.current?.querySelectorAll('g[id]').forEach((g) => g.classList.toggle('selected', g.id === selected))
  }, [selected, svg.data])

  const current: Model | undefined = model.data?.[view]

  return (
    <div className="canvas-layout">
      <section>
        <div className="toolbar">
          <div className="segmented">
            {(['context', 'container'] as const).map((v) => (
              <button key={v} className={view === v ? 'active' : ''} onClick={() => setView(v)}>
                {v === 'context' ? 'System context' : 'Containers'}
              </button>
            ))}
          </div>
          <button onClick={() => setFit(!fit)}>{fit ? 'Actual size' : 'Fit to width'}</button>
          <span className="muted small">
            Click an element to inspect it. <i>Italic</i> text was written by the model.
          </span>
        </div>
        {svg.error && <Failure error={svg.error} />}
        {svg.loading && <Loading />}
        {svg.data && (
          <div
            ref={ref}
            className={`canvas ${fit ? 'fit' : ''}`}
            onClick={(e) => setSelected((e.target as Element).closest('g[id]')?.id ?? null)}
            dangerouslySetInnerHTML={{ __html: svg.data }}
          />
        )}
      </section>
      <aside className="inspector">
        {model.error && <Failure error={model.error} />}
        {current && <Inspector model={current} id={selected} onSelect={setSelected} />}
      </aside>
    </div>
  )
}

/** Every value of the selected element, each with the file and line it came from. */
function Inspector({ model, id, onSelect }: { model: Model; id: string | null; onSelect: (id: string) => void }) {
  const nodes = model.nodes ?? []
  const edges = model.edges ?? []
  const node = nodes.find((n) => n.id === id)
  const name = (nid: string) => nodes.find((n) => n.id === nid)?.name ?? nid

  if (!node) {
    return (
      <>
        <h2>Elements</h2>
        <p className="muted small">
          {nodes.length} elements, {edges.length} relationships in this view.
        </p>
        <ul className="plain">
          {nodes.map((n) => (
            <li key={n.id}>
              <button className="link" onClick={() => onSelect(n.id)}>
                {n.name}
              </button>{' '}
              <span className="muted small">{n.kind}</span>
            </li>
          ))}
        </ul>
      </>
    )
  }

  const out = edges.filter((e) => e.from === node.id)
  const into = edges.filter((e) => e.to === node.id)

  return (
    <>
      <h2>{node.name}</h2>
      <p className="muted small mono">{node.id}</p>
      <dl className="facts">
        <Fact label="Name" value={node.name} prov={node.name_provenance ?? node.provenance} />
        <Fact label="Kind" value={node.kind} />
        <Fact label="Technology" value={node.technology} prov={node.technology_provenance} />
        <Fact label="Description" value={node.description} prov={node.description_provenance} />
        <Fact
          label="Evidence"
          value={node.evidence === 'declared' ? 'declared — the repository defines it' : 'referenced — only named by something else'}
        />
        {node.parent && <Fact label="Inside" value={name(node.parent)} />}
        {!!node.networks?.length && <Fact label="Networks" value={node.networks.join(', ')} />}
        <Fact label="Declared at" prov={node.provenance} />
      </dl>

      <Relationships title="Uses" edges={out} other={(e) => e.to} name={name} onSelect={onSelect} />
      <Relationships title="Used by" edges={into} other={(e) => e.from} name={name} onSelect={onSelect} />
    </>
  )
}

function Fact({ label, value, prov }: { label: string; value?: string; prov?: Node['provenance'] }) {
  return (
    <>
      <dt>{label}</dt>
      <dd>
        {value ? <div>{value}</div> : prov ? null : <div className="muted">—</div>}
        {prov && (value || label === 'Declared at') && <Prov p={prov} />}
      </dd>
    </>
  )
}

function Relationships(props: {
  title: string
  edges: Edge[]
  other: (e: Edge) => string
  name: (id: string) => string
  onSelect: (id: string) => void
}) {
  if (!props.edges.length) return null
  return (
    <>
      <h3>{props.title}</h3>
      <ul className="relations">
        {props.edges.map((e) => (
          <li key={`${e.from}>${e.to}`}>
            <button className="link" onClick={() => props.onSelect(props.other(e))}>
              {props.name(props.other(e))}
            </button>
            {e.label && (
              <div className={e.label_provenance?.origin === 'model' ? 'italic' : ''}>“{e.label}”</div>
            )}
            {e.technology && <div className="muted small">{e.technology}</div>}
            {e.label && e.label_provenance && (
              <div className="small">
                label: <Prov p={e.label_provenance} />
              </div>
            )}
            <div className="small">
              evidence:{' '}
              {(e.provenance ?? []).map((p, i) => (
                <Prov key={i} p={p} />
              ))}
            </div>
          </li>
        ))}
      </ul>
    </>
  )
}
