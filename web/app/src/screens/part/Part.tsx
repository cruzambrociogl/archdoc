import { useEffect, useMemo } from 'react'
import type { ModelResponse, SceneResponse } from '../../api'
import { useApi } from '../../api'
import type { Route } from '../../route'
import { Cite } from '../../ui/Cite'
import { Failure, Loading } from '../../ui/marks'
import { ViewDiagram } from '../explorer/ViewDiagram'
import type { Way } from './reach'
import { askable, highlight, reach, rows, sentence, ways } from './reach'
import type { Breakage } from './down'
import { downHighlight, downSentence, ordered } from './down'
import { addressOf, nameOf } from '../usage/faces'
import { TruthMark } from '../../ui/marks'

type Go = (r: Partial<Route>, o?: { keep?: boolean; replace?: boolean }) => void

type Question = Way | 'down'
const questions: Question[] = [...ways, 'down']
const tabs: Record<Question, string> = { touches: 'What it touches', depends: 'What depends on it', down: 'What breaks without it' }
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
 * What breaks without it is the engine's (/api/breaks): which ways in stop, worked out from the
 * tables each reads and writes and the jobs it queues, each at its line.
 */
export function Part(props: { route: Route; go: Go; version: number | null; onNames?: (names: Record<string, string>) => void }) {
  const { route, go } = props
  const subject = route.focus ?? ''
  const question: Question = route.question ?? 'touches'
  const way: Way = question === 'down' ? 'touches' : question
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

  const move = (patch: Partial<Route>, replace = false) => go({ screen: 'part', focus: subject, question, q: route.q, ...patch }, { replace })
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
          {questions.map((w) => (
            <button key={w} role="tab" aria-selected={question === w} className={`lens-btn ${question === w ? 'on' : ''}`} onClick={() => move({ question: w, all: undefined })}>
              {tabs[w]}
            </button>
          ))}
        </div>
      </div>

      {question === 'down' ? (
        <Down subject={subject} name={name} scene={scene.data} route={route} version={props.version} move={move} ask={ask} />
      ) : (
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
      )}
    </div>
  )
}

/** A list stops at five a part, with all on asking (rule 15). */
const FIRST = 5

/**
 * What breaks without it (app-design-screens §6, settled points 4 and 5): the part down, hatched and
 * tagged; what depends on it, tagged; and the ways in that stop, each with why and its line. Proven:
 * a rule over proven facts, and the band says what it was worked out from. Where how a part depends
 * is not recorded, it says so.
 */
function Down(props: {
  subject: string
  name: (id: string) => string
  scene: SceneResponse
  route: Route
  version: number | null
  move: (patch: Partial<Route>, replace?: boolean) => void
  ask: (id: string) => void
}) {
  const v = props.version ? `&version=${props.version}` : ''
  const b = useApi<Breakage>(`/api/breaks?part=${encodeURIComponent(props.subject)}${v}`)
  const m = useApi<ModelResponse>(`/api/model${props.version ? `?version=${props.version}` : ''}`)
  const entries = useMemo(() => new Map((m.data?.model.entries ?? []).map((e) => [e.id, e])), [m.data])
  if (b.error) return <Failure error={b.error} />
  if (!b.data || !m.data) return <Loading what="what breaks" />
  const { name } = props
  const known = b.data.dependents.filter((d) => d.known && !d.how.startsWith('a person'))
  const unknown = b.data.dependents.filter((d) => !d.known)
  const all = !!props.route.all
  const stopped = known.reduce((t, d) => t + d.stops.length, 0)

  return (
    <div className="part-body">
      <div className="band">
        <div className="band-kind">One part · an archdoc view</div>
        <h1 className="band-title">What breaks if {name(props.subject)} is down?</h1>
        <p className="band-line">{downSentence(b.data, name)}</p>
        <p className="band-meta">
          <TruthMark state="proven" /> {b.data.basis} Select another part on the diagram to ask about it.
        </p>
      </div>

      <ViewDiagram scene={props.scene} subject={props.subject} find={props.route.q ?? ''} highlight={downHighlight(b.data)} onPick={props.ask} onFind={(q) => props.move({ q: q || undefined }, true)} />

      {known.map((d) => (
        <section key={d.id} className="part-list" aria-label={`What stops in ${name(d.id)}`}>
          <div className="part-list-head">
            <h2>
              Stops · in {name(d.id)}
            </h2>
            <span className="muted small">
              {d.stops.length} · {d.how}
            </span>
          </div>
          {d.stops.length === 0 && <p className="muted">None of its ways in reach it.</p>}
          {(all ? ordered(d.stops) : ordered(d.stops).slice(0, FIRST)).map((e) => {
            const en = entries.get(e.entry)
            return (
              <div key={e.entry} className="part-row">
                <div className="part-row-main">
                  <span className="part-row-name">{en ? nameOf(en) : e.entry}</span>
                  <span className="part-tag direct">stops</span>
                </div>
                {en?.summary && <div className="part-row-what mono">{addressOf(en)}</div>}
                <div className="part-row-how">
                  {e.why.charAt(0).toUpperCase() + e.why.slice(1)} <Cite p={{ file: e.cite.file, line: e.cite.line }} compact />
                </div>
              </div>
            )
          })}
          {d.stops.length > FIRST && (
            <button className="link-btn part-all" onClick={() => props.move({ all: all ? undefined : '1' }, true)}>
              {all ? 'Show the first five' : `Show all ${d.stops.length}`}
            </button>
          )}
          {(d.unaffected > 0 || d.unfollowed > 0) && (
            <p className="part-note">
              {d.unaffected > 0 && <>Not affected: {d.unaffected} of its ways in reach nothing of {name(props.subject)}. </>}
              {d.unfollowed > 0 && <>Not known: {d.unfollowed} weren’t followed by the engine, so can’t be told.</>}
            </p>
          )}
        </section>
      ))}

      {unknown.length > 0 && (
        <section className="part-list" aria-label="Depends on it, how not recorded">
          <div className="part-list-head">
            <h2>Depends on it · which of its work, not recorded</h2>
            <span className="muted small">{unknown.length}</span>
          </div>
          {unknown.map((d) => (
            <div key={d.id} className="part-row">
              <div className="part-row-main">
                <button className="part-row-name" onClick={() => props.ask(d.id)}>
                  {name(d.id)}
                </button>
                <span className="part-tag indirect">not known</span>
              </div>
              <div className="part-row-how">
                It {d.how}, but which of its ways in make the call isn’t recorded: the call’s address is set when it runs, or the arrow comes from configuration.
              </div>
              <div className="part-row-cites">
                {(d.provenance ?? []).map((p, i) => (
                  <Cite key={i} p={p} compact />
                ))}
              </div>
            </div>
          ))}
        </section>
      )}

      {b.data.through.length > 0 && (
        <section className="part-list" aria-label="Through another part">
          <div className="part-list-head">
            <h2>Through another part</h2>
            <span className="muted small">{b.data.through.length}</span>
          </div>
          <p className="part-note">
            {b.data.through
              .map((t) => `${name(t.id)} through ${name(t.via)}`)
              .join('; ')}
            . What of theirs stops depends on which of {stopped ? 'those ways in' : 'its ways in'} they use, which the code doesn’t say.
          </p>
        </section>
      )}
    </div>
  )
}
