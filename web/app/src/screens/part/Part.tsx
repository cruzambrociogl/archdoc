import { useEffect, useMemo } from 'react'
import type { SceneResponse } from '../../api'
import { useApi } from '../../api'
import type { Route } from '../../route'
import { Cite } from '../../ui/Cite'
import { Failure, Loading } from '../../ui/marks'
import { ViewDiagram } from '../explorer/ViewDiagram'
import type { Way } from './reach'
import { askable, highlight, reach, rows, sentence, ways } from './reach'

type Go = (r: Partial<Route>, o?: { keep?: boolean; replace?: boolean }) => void

const tabs: Record<Way, string> = { touches: 'What it touches', depends: 'What depends on it' }
const titles: Record<Way, (name: string) => string> = {
  touches: (n) => `What does ${n} touch?`,
  depends: (n) => `What depends on ${n}?`,
}

/**
 * One part, and the questions about it (platform-surface §17.3; app-design-screens §6). An archdoc
 * view drawn on Apps and services — the same boxes in the same places — with what answers the
 * question lit and the rest dimmed. Every part reached is listed with how: the arrow and the line
 * that proves it, or the part it is reached through. Selecting another box changes the subject and
 * keeps the question.
 *
 * What breaks without it waits for the engine to record, with its proof, whether each call waits for
 * its answer or only queues work; it is not shown until then.
 */
export function Part(props: { route: Route; go: Go; version: number | null; onNames?: (names: Record<string, string>) => void }) {
  const { route, go } = props
  const subject = route.focus ?? ''
  const way: Way = route.question ?? 'touches'
  // Every external system, not only the main ones: reach follows every arrow there is.
  const scene = useApi<SceneResponse>(`/api/scene?view=container:all${props.version ? `&version=${props.version}` : ''}`)
  const model = scene.data?.model
  const nodes = useMemo(() => new Map((model?.nodes ?? []).map((n) => [n.id, n])), [model])
  const name = (id: string) => nodes.get(id)?.name ?? id.replace(/^[a-z]+:/, '')

  const onNames = props.onNames
  useEffect(() => {
    if (!model || !onNames) return
    const names: Record<string, string> = {}
    for (const n of model.nodes ?? []) names[n.id] = n.name
    onNames(names)
  }, [model, onNames])

  const move = (patch: Partial<Route>, replace = false) => go({ screen: 'part', focus: subject, question: way, q: route.q, ...patch }, { replace })
  const ask = (id: string) => askable(nodes.get(id)) && move({ focus: id, q: undefined })

  if (scene.error) return <Failure error={scene.error} />
  if (!scene.data || !model) return <Loading what="the part" />
  const node = nodes.get(subject)
  if (!node || !askable(node)) {
    // A person is on the diagram but is not a part of the system; anything else is not in this version.
    return (
      <div className="explorer-state reading">
        <div className="page-q">One part</div>
        <p className="lede">
          {node
            ? `${node.name} is a person who uses the system, not a part of it. Ask about one of the parts on Apps and services.`
            : `There’s no part called “${subject}” on Apps and services in this version.`}
        </p>
        <button className="link-btn" onClick={() => go({ screen: 'architecture', level: 'container' })}>
          Apps and services →
        </button>
      </div>
    )
  }

  const reached = reach(model.edges ?? [], subject, way)
  const list = rows(reached, name)
  const n = name(subject)

  return (
    <div className="part-view">
      <div className="explorer-toolbar part-bar">
        <div className="map-group" role="tablist" aria-label="The question">
          {ways.map((w) => (
            <button key={w} role="tab" aria-selected={way === w} className={`lens-btn ${way === w ? 'on' : ''}`} onClick={() => move({ question: w })}>
              {tabs[w]}
            </button>
          ))}
        </div>
      </div>

      <div className="part-body">
        <div className="band">
          <div className="band-kind">One part · an archdoc view</div>
          <h1 className="band-title">{titles[way](n)}</h1>
          <p className="band-line">{sentence(way, subject, reached, name)}</p>
          <p className="band-meta">Select another part on the diagram to ask about it.</p>
        </div>

        <ViewDiagram scene={scene.data} subject={subject} find={route.q ?? ''} highlight={highlight(subject, reached, name)} onPick={ask} onFind={(q) => move({ q: q || undefined }, true)} />

        <section className="part-list" aria-label={way === 'touches' ? 'How it reaches each one' : 'How each one reaches it'}>
          <div className="part-list-head">
            <h2>{way === 'touches' ? 'How it reaches each one' : 'How each one reaches it'}</h2>
            {list.length > 0 && <span className="muted small">{list.length === 1 ? '1 part' : `${list.length} parts`}</span>}
          </div>
          {list.length === 0 && <p className="muted">Nothing to list.</p>}
          {list.map((r) => {
            const target = nodes.get(r.id)
            const label = r.edges.map((e) => [e.label, e.technology].filter(Boolean).join(' · ')).filter(Boolean)
            return (
              <div key={r.id} className="part-row">
                <div className="part-row-main">
                  {askable(target) ? (
                    <button className="part-row-name" onClick={() => ask(r.id)} title={`Ask about ${name(r.id)}`}>
                      {name(r.id)}
                    </button>
                  ) : (
                    <span className="part-row-name">{name(r.id)}</span>
                  )}
                  <span className={`part-tag ${r.via ? 'indirect' : 'direct'}`}>{r.via ? 'indirect' : 'direct'}</span>
                </div>
                <div className="part-row-how">
                  {r.via ? `Through ${name(r.via)}` : label.length ? label.join('; ') : 'An arrow with no words'}
                </div>
                {!r.via && (
                  <div className="part-row-cites">
                    {r.edges.flatMap((e) => e.provenance ?? []).map((p, i) => (
                      <Cite key={i} p={p} compact />
                    ))}
                  </div>
                )}
              </div>
            )
          })}
        </section>
      </div>
    </div>
  )
}
