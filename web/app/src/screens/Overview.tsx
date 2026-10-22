import { useState } from 'react'
import type { CoverageResponse, Model, ModelResponse, Node, Summary, Version } from '../api'
import { useApi } from '../api'
import type { Route } from '../route'
import { Cite } from '../ui/Cite'
import { kindStyle } from '../ui/kinds'
import { Failure, Loading, TruthMark } from '../ui/marks'
import type { Piece } from '../sentence'
import { namedIn, structureSentence } from '../sentence'

/**
 * The Overview (platform-surface §6.1): what is this system? One paragraph in plain words, one
 * picture, one sentence of trust — nothing else above the fold (rule 15). The paragraph's structure
 * is written by a template from the container graph, so it is proven; a purpose comes first only
 * when the repository states one or someone asked AI for one, and says which. The counts that used
 * to fill this page live in Reference and About these docs.
 */
export function Overview(props: { summary: Summary; versions: Version[]; version: number | null; go: (r: Partial<Route>) => void; coverage?: CoverageResponse }) {
  const q = props.version ? `?version=${props.version}` : ''
  const m = useApi<ModelResponse>(`/api/model${q}`)
  const [evidence, setEvidence] = useState(false)
  // The picture as a list (§4.8, a text alternative for every diagram): on request, and always for a screen reader.
  const [asList, setAsList] = useState(false)

  if (m.error) return <Failure error={m.error} />
  if (!m.data) return <Loading q="What is this?" what="its parts and how they connect" />
  const { model, container, context } = m.data
  const system = (context.nodes ?? []).find((n) => n.kind === 'system')
  const purpose = system?.description
  const purposeByAI = system?.description_provenance?.origin === 'model'

  // Did AI write anything shown in these docs? Then the trust sentence says so.
  const byAI =
    purposeByAI ||
    (model.explanations ?? []).length > 0 ||
    (model.nodes ?? []).some((n) => n.description_provenance?.origin === 'model') ||
    (model.edges ?? []).some((e) => e.label_provenance?.origin === 'model')
  const unresolved = (model.unresolved ?? []).length

  const sentences = props.summary.tiny ? [] : structureSentence(props.summary.name, container)
  const cited = namedIn(sentences)
  const open = (id: string) => props.go({ screen: 'explorer', level: props.summary.tiny ? 'context' : 'container', focus: id })
  const parts = (container.nodes ?? []).filter((n) => ['application', 'datastore', 'queue'].includes(n.kind))

  return (
    <div className="reading overview">
      <h1 className="visually-hidden">{props.summary.name}</h1>
      <div className="page-q">What is this?</div>

      {props.summary.tiny && props.summary.story ? (
        // A small project is told, not counted: what it is, how it is used, what it reaches.
        <div className="story">
          {props.summary.story.map((l, i) => (
            <p key={i} className={i === 0 ? 'overview-answer' : 'story-line'}>
              {l.interpreted && <TruthMark state="interpreted" />}{' '}
              <span className={l.interpreted ? 'interp-sentence' : ''}>{l.text.split('`').map((part, j) => (j % 2 ? <span key={j} className="mono">{part}</span> : part))}</span>{' '}
              {l.provenance?.file && <Cite p={l.provenance} compact />}
            </p>
          ))}
        </div>
      ) : (
        <>
          <p className="overview-answer">
            {/* A manifest's description often has no full stop; the next sentence needs one. */}
            {purpose && <span className={purposeByAI ? 'interp-sentence' : ''}>{/[.!?]$/.test(purpose) ? purpose : `${purpose}.`} </span>}
            {sentences.map((s, i) => (
              <span key={i}>
                <Sentence pieces={s} open={open} />{' '}
              </span>
            ))}
          </p>
          <div className="overview-source">
            {purpose && purposeByAI && (
              <span className="written-by-ai">
                <TruthMark state="interpreted" /> The first sentence is written by AI from these facts
              </span>
            )}
            {purpose && !purposeByAI && system?.description_provenance && (
              <span>
                The first sentence is the repository's own · <Cite p={system.description_provenance} compact />
              </span>
            )}
            {cited.length > 0 && (
              <button className="evidence-toggle" aria-expanded={evidence} onClick={() => setEvidence((e) => !e)}>
                <span className="caret">{evidence ? '▾' : '▸'}</span> Evidence <span className="muted">{cited.length} sources</span>
              </button>
            )}
          </div>
          {evidence && (
            <ul className="evidence-list">
              {cited.map((n) => (
                <li key={n.id}>
                  <button className="part-link" onClick={() => open(n.id)}>
                    {n.name}
                  </button>
                  <span className="muted small"> {kindStyle(n.kind).label}</span> <Cite p={n.provenance} compact />
                </li>
              ))}
            </ul>
          )}
        </>
      )}

      {/* One picture: the most useful level, a way into the Map (§6.1, item 2). */}
      {!props.summary.tiny && parts.length > 0 && !asList && (
        <button className="thumb overview-picture" onClick={() => props.go({ screen: 'explorer', level: 'container' })} aria-label="Apps and services — open the map">
          <div className="thumb-boundary">
            {parts.slice(0, 12).map((n) => (
              <div key={n.id} className={`thumb-box tile-${kindStyle(n.kind).hue}`}>
                {n.name}
              </div>
            ))}
          </div>
          <div className="thumb-foot">
            <span className="mono muted small">apps and services{parts.length > 12 ? ` · ${parts.length - 12} more on the map` : ''}</span>
            <span className="btn btn-ink">Open the map →</span>
          </div>
        </button>
      )}

      {!props.summary.tiny && parts.length > 0 && (
        <>
          <button className="link-btn picture-toggle" aria-expanded={asList} onClick={() => setAsList((l) => !l)}>
            {asList ? 'Show the picture' : 'Read the picture as a list'}
          </button>
          <PartsList parts={parts} graph={container} open={asList ? open : undefined} />
        </>
      )}

      {/* Trust in one sentence — the one second sentence the density budget allows. It opens About these docs. */}
      <button className="trust-sentence" onClick={() => props.go({ screen: 'about' })}>
        <TruthMark state="unresolved" />
        <span>
          {byAI ? 'Every part and connection here is proven from your code; the plain sentences are written by AI on top of them.' : 'Everything here is proven from your code.'}{' '}
          {unresolved ? `${unresolved} ${unresolved === 1 ? 'thing' : 'things'} couldn't be resolved.` : 'Nothing was left unresolved.'}
          <span className="trust-more"> About these docs →</span>
        </span>
      </button>
    </div>
  )
}

/** A sentence whose named parts open on the Map. */
function Sentence({ pieces, open }: { pieces: Piece[]; open: (id: string) => void }) {
  return (
    <>
      {pieces.map((p, i) =>
        typeof p === 'string' ? (
          <span key={i}>{p}</span>
        ) : (
          <button key={i} className="part-link" onClick={() => open(p.node.id)} title={`${p.node.name} — on the map`}>
            {p.node.name}
          </button>
        ),
      )}
    </>
  )
}

/**
 * The picture's parts and what each calls, as a list: shown when asked for, and otherwise there
 * for a screen reader alone, without anything to focus.
 */
function PartsList({ parts, graph, open }: { parts: Node[]; graph: Model; open?: (id: string) => void }) {
  const byId = new Map((graph.nodes ?? []).map((n) => [n.id, n]))
  const calls = (id: string) => [...new Set((graph.edges ?? []).filter((e) => e.from === id && byId.has(e.to)).map((e) => byId.get(e.to)!.name))]
  return (
    <ul className={open ? 'parts-list' : 'visually-hidden'} aria-label="Apps and services, as a list">
      {parts.map((n) => {
        const to = calls(n.id)
        return (
          <li key={n.id}>
            {open ? (
              <button className="part-link" onClick={() => open(n.id)}>
                {n.name}
              </button>
            ) : (
              n.name
            )}{' '}
            <span className="muted small">{kindStyle(n.kind).label}{n.technology ? ` · ${n.technology}` : ''}</span>
            {n.description && <div>{n.description}</div>}
            {to.length > 0 && <div className="muted">Calls {to.join(', ')}.</div>}
          </li>
        )
      })}
    </ul>
  )
}
