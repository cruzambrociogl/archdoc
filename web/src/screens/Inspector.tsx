import { useState } from 'react'
import type { Change, Edge, Model, Node, Provenance } from '../api'
import { Cite } from '../ui/Cite'
import { KindTile } from '../ui/KindTile'
import { kindStyle } from '../ui/kinds'
import { Eyebrow, TruthChip, TruthMark } from '../ui/marks'

/**
 * An element's passport (surface-spec §5.3): what it is, how we know, and what touches it. Every
 * value carries its own citation. The element itself is always proven — existence comes only from
 * extraction — so only its values can be interpreted, and they are marked one by one.
 */
type Changes = { from: number; mark?: 'added' | 'changed' | 'words'; list: Change[] }

export function Inspector({ model, id, onSelect, changes }: { model: Model; id: string | null; onSelect: (id: string) => void; changes?: Changes }) {
  const nodes = model.nodes ?? []
  const edges = model.edges ?? []
  const node = nodes.find((n) => n.id === id)
  const edge = node ? undefined : edges.find((e) => `${e.from}>${e.to}` === id)
  const what = node ? kindStyle(node.kind).label.toLowerCase() : edge ? 'relationship' : `${nodes.length} elements`

  return (
    <aside className="inspector">
      <div className="inspector-head">
        <Eyebrow>Inspector</Eyebrow>
        <span className="mono muted small">{what}</span>
      </div>
      {(node || edge) && changes?.mark && <ChangeNote changes={changes} />}
      {node ? (
        <Passport node={node} nodes={nodes} edges={edges} onSelect={onSelect} />
      ) : edge ? (
        <EdgePassport edge={edge} nodes={nodes} onSelect={onSelect} />
      ) : (
        <Index nodes={nodes} edges={edges} onSelect={onSelect} />
      )}
    </aside>
  )
}

/** What changed about the selection since the compared version. */
function ChangeNote({ changes }: { changes: Changes }) {
  const label = changes.mark === 'added' ? 'Appeared' : changes.mark === 'words' ? 'Words only' : 'Changed'
  return (
    <div className="change-note">
      <div className="change-note-head">
        <span className={`delta-mark inline ${changes.mark === 'words' ? 'hollow' : ''}`}>{changes.mark === 'added' ? '+' : '±'}</span>
        <span className="strong">{label}</span>
        <span className="mono muted small">since v{changes.from}</span>
      </div>
      {changes.list.map((c, i) => (
        <div key={i} className="change-note-row">
          <span className="fact-key">{c.field}</span>
          <span className="mono small">
            {c.before || '—'} → {c.after || '—'}
          </span>
        </div>
      ))}
    </div>
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
    ['code', node.dir ? (node.dir === '.' ? 'the repository root' : `${node.dir}/`) : undefined, cited(node.dir_provenance)],
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

/** An arrow is first-class (F-55): its ends, what it says, and every line that proves it. */
function EdgePassport({ edge: e, nodes, onSelect }: { edge: Edge; nodes: Node[]; onSelect: (id: string) => void }) {
  const name = (id: string) => nodes.find((n) => n.id === id)?.name ?? id
  const interp = e.label_provenance?.origin === 'model'
  const sources = e.provenance ?? []
  return (
    <>
      <div className="passport-head">
        <div className="passport-name">
          <button className="relation-other" onClick={() => onSelect(e.from)}>
            {name(e.from)}
          </button>{' '}
          <span className="muted">→</span>{' '}
          <button className="relation-other" onClick={() => onSelect(e.to)}>
            {name(e.to)}
          </button>
        </div>
        <div className="passport-chips">
          <TruthChip state="proven" />
          {e.traffic ? (
            <span className="kind-chip hue-grey" title="A host or port was configured, so something flows">
              traffic
            </span>
          ) : (
            <span className="kind-chip hue-grey dashed" title="depends_on: one service starts before the other — not evidence that anything flows">
              start order only
            </span>
          )}
        </div>
      </div>
      <section className="inspector-section">
        <Eyebrow>Facts</Eyebrow>
        {e.label && (
          <div className="fact">
            <div className="fact-row">
              <span className="fact-key">label</span>
              <span className={`fact-value ${interp ? 'interpreted' : ''}`}>
                {interp && <TruthMark state="interpreted" />} {e.label}
              </span>
            </div>
            {cited(e.label_provenance) && (
              <div className="fact-cite">
                <Cite p={e.label_provenance} compact />
              </div>
            )}
          </div>
        )}
        {e.technology && (
          <div className="fact">
            <div className="fact-row">
              <span className="fact-key">protocol</span>
              <span className="fact-value">{e.technology}</span>
            </div>
          </div>
        )}
        <div className="fact">
          <div className="fact-row">
            <span className="fact-key">{sources.length > 1 ? `proven by ${sources.length}` : 'proven by'}</span>
            <span className="fact-value relation-cites">
              {sources.length ? sources.map((p, i) => <Cite key={i} p={p} />) : <span className="cite cite-missing">no source recorded</span>}
            </span>
          </div>
        </div>
      </section>
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
