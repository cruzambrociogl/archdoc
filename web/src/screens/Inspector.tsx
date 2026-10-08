import { useState } from 'react'
import type { Change, Edge, Entry, Explanation, Model, Node, Provenance } from '../api'
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

export function Inspector({
  model,
  id,
  onSelect,
  changes,
  opens,
  onOpen,
  holds,
  onOpenData,
}: {
  model: Model
  id: string | null
  onSelect: (id: string) => void
  changes?: Changes
  /** Containers whose code was read, and how many components each opens onto. */
  opens?: Map<string, number>
  onOpen?: (id: string) => void
  /** Containers whose code declares tables, and how many. */
  holds?: Map<string, number>
  onOpenData?: (id: string) => void
}) {
  const nodes = model.nodes ?? []
  const edges = model.edges ?? []
  const node = nodes.find((n) => n.id === id)
  const edge = node ? undefined : edges.find((e) => `${e.from}>${e.to}` === id)
  const parts = nodes.length > 0 && nodes.every((n) => n.kind === 'component' || n.kind === 'table')
  const what = node ? kindStyle(node.kind).label.toLowerCase() : edge ? (edge.weight ? 'import' : 'relationship') : `${nodes.length} ${parts ? (nodes[0].kind === 'table' ? 'tables' : 'components') : 'elements'}`

  return (
    <aside className="inspector">
      <div className="inspector-head">
        <Eyebrow>Inspector</Eyebrow>
        <span className="mono muted small">{what}</span>
      </div>
      {(node || edge) && changes?.mark && <ChangeNote changes={changes} />}
      {node ? (
        <Passport
          node={node}
          nodes={nodes}
          edges={edges}
          explanation={(model.explanations ?? []).find((x) => x.element === node.id)}
          entries={(model.entries ?? []).filter((e) => e.component === node.id)}
          onSelect={onSelect}
          container={model.name}
          opens={opens?.get(node.id)}
          onOpen={onOpen}
          holds={holds?.get(node.id)}
          onOpenData={onOpenData}
        />
      ) : edge ? (
        <EdgePassport edge={edge} nodes={nodes} onSelect={onSelect} />
      ) : (
        <Index nodes={nodes} edges={edges} onSelect={onSelect} parts={parts} />
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

function Index({ nodes, edges, onSelect, parts }: { nodes: Node[]; edges: Edge[]; onSelect: (id: string) => void; parts: boolean }) {
  return (
    <div className="inspector-body">
      <p className="inspector-hint">Select a box. Every value it shows carries the line that proves it.</p>
      <Eyebrow>
        {parts
          ? nodes[0].kind === 'table'
            ? `${nodes.length} tables · ${edges.length} foreign keys`
            : `${nodes.length} components · ${edges.length} uses`
          : `${nodes.length} elements · ${edges.length} relationships`}
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

function Passport({
  node,
  nodes,
  edges,
  explanation,
  entries,
  onSelect,
  container,
  opens,
  onOpen,
  holds,
  onOpenData,
}: {
  node: Node
  nodes: Node[]
  edges: Edge[]
  explanation?: Explanation
  entries: Entry[]
  onSelect: (id: string) => void
  /** The view's name: in a component view, the container every component is inside. */
  container: string
  opens?: number
  onOpen?: (id: string) => void
  holds?: number
  onOpenData?: (id: string) => void
}) {
  const name = (id: string) => nodes.find((n) => n.id === id)?.name ?? (id === node.parent ? container : id)
  const component = node.kind === 'component'
  const k = kindStyle(node.kind)
  const out = edges.filter((e) => e.from === node.id)
  const into = edges.filter((e) => e.to === node.id)
  const [copied, setCopied] = useState(false)

  // A value with an empty provenance came with the declaration itself, so it cites that line.
  const facts: [string, string | undefined, Provenance | undefined][] = [
    ['name', node.name, cited(node.name_provenance) ?? node.provenance],
    ['technology', node.technology, cited(node.technology_provenance) ?? node.provenance],
    ['description', node.description, cited(node.description_provenance) ?? node.provenance],
    ['code', node.dir ? (node.dir === '.' ? 'the repository root' : node.kind === 'table' || (component && node.files?.length === 1) ? node.dir : `${node.dir}/`) : undefined, cited(node.dir_provenance)],
    ['size', component && node.files ? `${node.files.length} ${node.files.length === 1 ? 'file' : 'files'} · ${(node.lines ?? 0).toLocaleString()} lines` : undefined, undefined],
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

      {explanation && (
        <section className="inspector-section">
          <Eyebrow>What it does · interpreted</Eyebrow>
          {explanation.claims.map((c, i) => (
            <div key={i} className="claim">
              <p className="interpreted">
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
            Written by {explanation.provenance.note} from the facts archdoc read; every sentence cites them.
            {explanation.stale && (
              <>
                {' '}
                <TruthMark state="unresolved" /> Those facts have changed since: this is the last answer, kept until{' '}
                <span className="mono">generate --explain</span> asks again.
              </>
            )}
          </p>
        </section>
      )}

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
            <span className="fact-key">{component ? 'found at' : 'declared at'}</span>
            <span className="fact-value">
              <Cite p={node.provenance} />
            </span>
          </div>
        </div>
        {opens !== undefined && onOpen && (
          <div className="fact">
            <div className="fact-row">
              <span className="fact-key">inside</span>
              <span className="fact-value">
                <button className="text-link" onClick={() => onOpen(node.id)}>
                  {node.kind === 'system' ? `${opens} containers ⤵` : `${opens} components, from its code ⤵`}
                </button>
              </span>
            </div>
          </div>
        )}
        {holds !== undefined && onOpenData && (
          <div className="fact">
            <div className="fact-row">
              <span className="fact-key">data</span>
              <span className="fact-value">
                <button className="text-link" onClick={() => onOpenData(node.id)}>
                  {holds} tables its code declares ⤵
                </button>
              </span>
            </div>
          </div>
        )}
      </section>

      {node.columns && node.columns.length > 0 && (
        <section className="inspector-section">
          <Eyebrow>{node.columns.length} columns</Eyebrow>
          {node.columns.map((c) => (
            <div className="column-fact" key={c.name}>
              <div className="column-row">
                <span className="mono column-name">
                  {c.name}
                  {c.nullable ? '?' : ''}
                </span>
                <span className="column-type">
                  <span className="mono small">{[c.primary && 'PK', c.type].filter(Boolean).join(' ')}</span>
                  {c.references && (
                    <>
                      {' '}
                      <span className="muted small">→</span>{' '}
                      <button className="relation-other" onClick={() => onSelect(c.references!)}>
                        {name(c.references)}
                      </button>
                    </>
                  )}
                </span>
              </div>
              <div className="fact-cite">
                <Cite p={c.provenance} compact />
              </div>
            </div>
          ))}
        </section>
      )}

      {entries.length > 0 && (
        <section className="inspector-section">
          <Eyebrow>
            Handles {entries.length} {entries.length === 1 ? 'route' : 'routes'}
          </Eyebrow>
          <ul className="file-list">
            {entries.slice(0, 12).map((e) => (
              <li key={e.id} className="mono small">
                {e.kind === 'page' ? '' : `${e.method} `}
                {e.path}
              </li>
            ))}
            {entries.length > 12 && <li className="small muted">and {entries.length - 12} more on the Features screen</li>}
          </ul>
        </section>
      )}

      {component && node.files && node.files.length > 0 && (
        <section className="inspector-section">
          <details className="files">
            <summary>
              <Eyebrow>
                {node.files.length} {node.files.length === 1 ? 'file' : 'files'}
              </Eyebrow>
            </summary>
            <ul className="file-list">
              {node.files.map((f) => (
                <li key={f}>
                  <Cite p={{ file: f, line: 1 }} compact />
                </li>
              ))}
            </ul>
          </details>
        </section>
      )}

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
          {e.weight ? (
            <span className="kind-chip hue-grey" title="One part of the code imports the other">
              import
            </span>
          ) : e.traffic ? (
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
        {e.weight !== undefined && (
          <div className="fact">
            <div className="fact-row">
              <span className="fact-key">imports</span>
              <span className="fact-value">
                {e.weight} {e.weight === 1 ? 'import' : 'imports'}
                {sources.length < e.weight && <span className="muted"> · the first in each importing file is cited, at most ten</span>}
              </span>
            </div>
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
