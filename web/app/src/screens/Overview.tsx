import { Fragment, useState } from 'react'
import type { CoverageResponse, Model, ModelResponse, Node, Summary, Version } from '../api'
import { useApi } from '../api'
import type { Route } from '../route'
import { Cite } from '../ui/Cite'
import { kindStyle } from '../ui/kinds'
import { Failure, Loading, TruthMark } from '../ui/marks'
import type { Piece } from '../sentence'
import { namedIn, number, structureColumns, structureSentence } from '../sentence'
import { ReturnVisit } from './home/ReturnVisit'

/**
 * The Overview (platform-surface §6.1): what is this system? One paragraph in plain words, one
 * picture, one sentence of trust — nothing else above the fold (rule 15). The paragraph's structure
 * is written by a template from the container graph, so it is proven; a purpose comes first only
 * when the repository states one or someone asked AI for one, and says which. The counts that used
 * to fill this page live in Reference and About these docs.
 */
export function Overview(props: { summary: Summary; versions: Version[]; version: number | null; go: (r: Partial<Route>) => void; coverage?: CoverageResponse; seen?: number }) {
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
  const open = (id: string) => props.go({ screen: 'architecture', level: props.summary.tiny ? 'context' : 'container', focus: id })
  const parts = (container.nodes ?? []).filter((n) => ['application', 'datastore', 'queue'].includes(n.kind))

  return (
    <div className="reading overview">
      <h1 className="visually-hidden">{props.summary.name}</h1>
      {props.seen !== undefined && props.version === null && <ReturnVisit seen={props.seen} versions={props.versions} go={props.go} />}
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

      {/* One picture: how the parts fit, a way into the Map (§6.1, item 2). */}
      {!props.summary.tiny && parts.length > 0 && !asList && <HowItFits graph={container} open={open} openMap={() => props.go({ screen: 'architecture', level: 'container' })} />}

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
          <span className="trust-more"> What archdoc knows →</span>
        </span>
      </button>
    </div>
  )
}

// A column shows this many parts, and counts the rest; the Map has every one.
const SHOWN = 6

/** The parts in columns by who calls whom, each one a way onto the Map. Seen, not read: the list below is its text. */
function HowItFits({ graph, open, openMap }: { graph: Model; open: (id: string) => void; openMap: () => void }) {
  const { columns, arrows, outside } = structureColumns(graph)
  return (
    <div className="fit overview-picture">
      <div className="fit-cols" aria-hidden="true" style={{ gridTemplateColumns: columns.map(() => 'minmax(0, 1fr)').join(' 28px ') }}>
        {/* The titles are a row of their own, so they line up whatever each column holds. */}
        {columns.map((c, i) => (
          <Fragment key={c.title}>
            {i > 0 && <span />}
            <div className="fit-title">{c.title}</div>
          </Fragment>
        ))}
        {columns.map((c, i) => (
          <Fragment key={c.title}>
            {i > 0 && <span className={`fit-arrow ${arrows[i - 1] ? '' : 'fit-arrow-none'}`} />}
            <div className="fit-col" data-title={c.title}>
              {c.parts.slice(0, SHOWN).map((n) => {
                const k = kindStyle(n.kind)
                return (
                  <button key={n.id} className={`fit-box tile-${k.hue}`} tabIndex={-1} onClick={() => open(n.id)} title={`${n.name} · ${n.technology || k.label} — open in Architecture`}>
                    <svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinejoin="round" strokeLinecap="round">
                      <path d={k.icon} />
                    </svg>
                    <span>{n.name}</span>
                  </button>
                )
              })}
              {c.parts.length > SHOWN && <div className="fit-more">+ {c.parts.length - SHOWN} more in Architecture</div>}
            </div>
          </Fragment>
        ))}
      </div>
      <div className="fit-foot">
        <span>{outside > 0 && `It reaches ${number(outside)} outside ${outside === 1 ? 'service' : 'services'}.`}</span>
        <button className="btn btn-ink" onClick={openMap}>
          See the architecture →
        </button>
      </div>
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
          <button key={i} className="part-link" onClick={() => open(p.node.id)} title={`${p.node.name} — in Architecture`}>
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
