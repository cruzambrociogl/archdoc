import { useState } from 'react'
import type { Edge, Model, Node, Provenance } from '../api'
import { Cite } from '../ui/Cite'
import { KindTile } from '../ui/KindTile'
import { kindStyle } from '../ui/kinds'
import { Eyebrow, TruthChip, TruthMark } from '../ui/marks'

/**
 * An element's passport (surface-spec §5.3): what it is, how we know, and what touches it. Every
 * value carries its own citation. The element itself is always proven — existence comes only from
 * extraction — so only its values can be interpreted, and they are marked one by one.
 */
export function Inspector({ model, id, onSelect }: { model: Model; id: string | null; onSelect: (id: string) => void }) {
  const nodes = model.nodes ?? []
  const edges = model.edges ?? []
  const node = nodes.find((n) => n.id === id)

  return (
    <aside className="inspector">
      <div className="inspector-head">
        <Eyebrow>Inspector</Eyebrow>
        <span className="mono muted small">{node ? kindStyle(node.kind).label.toLowerCase() : `${nodes.length} elements`}</span>
      </div>
      {node ? <Passport node={node} nodes={nodes} edges={edges} onSelect={onSelect} /> : <Index nodes={nodes} edges={edges} onSelect={onSelect} />}
    </aside>
  )
}

function Index({ nodes, edges, onSelect }: { nodes: Node[]; edges: Edge[]; onSelect: (id: string) => void }) {
  return (
    <div className="inspector-body">
      <p className="inspector-hint">Select a box. Every value it shows carries the line that proves it.</p>
      <Eyebrow>
        {nodes.length} elements · {edges.length} relationships
      </Eyebrow>
      <ul className="index-list">
        {nodes.map((n) => (
          <li key={n.id}>
            <button onClick={() => onSelect(n.id)}>
              <KindTile kind={n.kind} referenced={n.evidence === 'referenced'} size={20} />
              <span className="index-name">{n.name}</span>
              <span className="index-kind">{kindStyle(n.kind).label}</span>
            </button>
          </li>
        ))}
      </ul>
    </div>
  )
}

function Passport({ node, nodes, edges, onSelect }: { node: Node; nodes: Node[]; edges: Edge[]; onSelect: (id: string) => void }) {
  const name = (id: string) => nodes.find((n) => n.id === id)?.name ?? id
  const k = kindStyle(node.kind)
  const out = edges.filter((e) => e.from === node.id)
  const into = edges.filter((e) => e.to === node.id)
  const [copied, setCopied] = useState(false)

  // A value with an empty provenance came with the declaration itself, so it cites that line.
  const facts: [string, string | undefined, Provenance | undefined][] = [
    ['name', node.name, cited(node.name_provenance) ?? node.provenance],
    ['technology', node.technology, cited(node.technology_provenance) ?? node.provenance],
    ['description', node.description, cited(node.description_provenance) ?? node.provenance],
    ['inside', node.parent ? name(node.parent) : undefined, undefined],
    ['networks', node.networks?.length ? node.networks.join(', ') : undefined, undefined],
  ]

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(location.href)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      /* clipboard refused — the address bar still holds the link */
    }
  }

  return (
    <>
      <div className="passport-head">
        <div className="passport-title">
          <KindTile kind={node.kind} referenced={node.evidence === 'referenced'} />
          <div className="passport-name-block">
            <div className="passport-name">{node.name}</div>
            <div className="passport-sub">{[k.label, node.technology].filter(Boolean).join(' · ')}</div>
          </div>
        </div>
        <div className="passport-chips">
          <span className={`kind-chip hue-${k.hue}`}>{k.label}</span>
          <TruthChip state="proven" />
          <span
            className={`kind-chip hue-grey ${node.evidence === 'referenced' ? 'dashed' : ''}`}
            title={node.evidence === 'declared' ? 'The repository defines it' : 'The repository only names it'}
          >
            {node.evidence}
          </span>
        </div>
      </div>

      <section className="inspector-section">
        <Eyebrow>Facts</Eyebrow>
        {facts
          .filter(([, v]) => v)
          .map(([key, value, prov]) => {
            const interp = prov?.origin === 'model'
            return (
              <div className="fact" key={key}>
                <div className="fact-row">
                  <span className="fact-key">{key}</span>
                  <span className={`fact-value ${interp ? 'interpreted' : ''}`}>
                    {interp && <TruthMark state="interpreted" />} {value}
                  </span>
                </div>
                {prov && (
                  <div className="fact-cite">
                    <Cite p={prov} compact />
                  </div>
                )}
              </div>
            )
          })}
        <div className="fact">
          <div className="fact-row">
            <span className="fact-key">declared at</span>
            <span className="fact-value">
              <Cite p={node.provenance} />
            </span>
          </div>
        </div>
      </section>

      {(out.length > 0 || into.length > 0) && (
        <section className="inspector-section">
          <Eyebrow>Relationships</Eyebrow>
          {out.map((e) => (
            <Relation key={`o${e.to}`} dir="→" other={e.to} edge={e} name={name} onSelect={onSelect} />
          ))}
          {into.map((e) => (
            <Relation key={`i${e.from}`} dir="←" other={e.from} edge={e} name={name} onSelect={onSelect} />
          ))}
        </section>
      )}

      <div className="inspector-actions">
        <button className="link-btn" onClick={copy}>
          {copied ? 'Link copied' : 'Copy link'}
        </button>
      </div>
    </>
  )
}

/** A provenance that names a source, or nothing. The engine sends an empty one for "same as the element". */
function cited(p?: Provenance): Provenance | undefined {
  return p && (p.file || p.note) ? p : undefined
}

function Relation(props: { dir: string; other: string; edge: Edge; name: (id: string) => string; onSelect: (id: string) => void }) {
  const { edge: e } = props
  const interp = e.label_provenance?.origin === 'model'
  return (
    <div className="relation">
      <span className="relation-dir">{props.dir}</span>
      <div className="relation-body">
        <button className="relation-other" onClick={() => props.onSelect(props.other)}>
          {props.name(props.other)}
        </button>
        {(e.label || e.technology) && (
          <div className="relation-meta">
            {e.label && (
              <span className={interp ? 'interpreted' : ''}>
                {interp && <TruthMark state="interpreted" />} {e.label}
              </span>
            )}
            {e.label && e.technology && ' · '}
            {e.technology}
          </div>
        )}
        <div className="relation-cites">
          {interp && cited(e.label_provenance) && <Cite p={e.label_provenance} compact />}
          {(e.provenance ?? []).map((p, i) => (
            <Cite key={i} p={p} compact />
          ))}
        </div>
      </div>
    </div>
  )
}
