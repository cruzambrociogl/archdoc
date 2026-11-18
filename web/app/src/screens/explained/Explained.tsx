import { useEffect, useMemo, useRef, useState } from 'react'
import type { Interpretation, Model, ModelResponse, Scenario, SceneResponse, Summary, Version } from '../../api'
import { useApi } from '../../api'
import type { Route } from '../../route'
import { Failure, Loading, TruthMark } from '../../ui/marks'
import { ViewDiagram } from '../explorer/ViewDiagram'
import { ReturnVisit } from '../home/ReturnVisit'
import type { Highlight } from '../explorer/scene'
import {
  boxesOfAll,
  featureFacts,
  citeTarget,
  featureAt,
  featureSlugs,
  scenarioAt,
  scenarioReach,
  scenarioSlugs,
  stepSplit,
  stepsLine,
} from '../../explain'

// The views that explain (platform-surface §16–17; app-design-screens §2, §8, §9): the tour of a
// first visit, the review, features by purpose, scenarios read three ways, and the C4 dynamic
// diagram. Each reads the explanation the engine stored and the facts it cites — nothing is drawn the
// explanation did not write — and every block a model wrote carries its mark.

type Go = (r: Partial<Route>, o?: { keep?: boolean; replace?: boolean }) => void

const q = (version: number | null) => (version ? `?version=${version}` : '')
const qa = (version: number | null) => (version ? `&version=${version}` : '')

/** A claim's mark, and the line that says what it is: written by AI from the facts, each one cited. */
export function AIMark({ line }: { line?: string }) {
  return (
    <span className="written-by-ai">
      <TruthMark state="interpreted" /> {line ?? 'Written by AI from the facts'}
    </span>
  )
}

/** A claim's citations, each opening what it names. An ID that names nothing here is left out. */
export function Cites({ ids, model, it, go }: { ids: string[]; model: Model; it: Interpretation; go: Go }) {
  const shown = ids.map((id) => ({ id, t: citeTarget(id, model, it) })).filter((c) => c.t)
  if (!shown.length) return null
  return (
    <span className="cites" aria-label="What this rests on">
      {shown.map(({ id, t }) => (
        <button key={id} className="cite-chip" title={id} onClick={() => go(t!.to)}>
          {t!.label}
        </button>
      ))}
    </span>
  )
}

/**
 * A version with no explanation (§4.8, the state without AI): why, in one sentence, and that what
 * archdoc proves stands without it. Locally, how to get one.
 */
export function NotExplained({ why, go, onFeatures }: { why?: string; go: Go; onFeatures?: boolean }) {
  const plain = !why || /not explained —/.test(why) ? 'This version is not explained.' : why[0].toUpperCase() + why.slice(1) + (why.endsWith('.') ? '' : '.')
  return (
    <div className="not-explained" role="note">
      <TruthMark state="proven" />
      <span>
        {plain} Everything here is proven from the code; the explanations written by AI are absent.
        {(!why || /archdoc interpret/.test(why)) && (
          <span className="muted">
            {' '}
            <span className="mono">archdoc interpret</span> explains it, for about a cent.
          </span>
        )}{' '}
        {!onFeatures && (
          <button className="link-btn" onClick={() => go({ screen: 'features' })}>
            What it does, from the facts →
          </button>
        )}
      </span>
    </div>
  )
}

function useFacts(version: number | null) {
  const m = useApi<ModelResponse>(`/api/model${q(version)}`)
  const scene = useApi<SceneResponse>(`/api/scene?view=container${qa(version)}`)
  return { model: m.data?.model, scene: scene.data, error: m.error ?? scene.error }
}

/** Lights the boxes of Apps and services a set of facts sits in; outlines none. */
function lit(ids: string[], model: Model): Highlight {
  return { lit: new Set(boxesOfAll(ids, model)) }
}

// ——— Home, a first visit: the tour ———

const tourParts = ['What is it?', 'What is it for?', 'What happens when…?', 'What should you know?']

/** Home's first visit (app-design-screens §2): the explanation in four parts, under a strip that follows the page. */
export function Tour(props: { it: Interpretation; summary: Summary; version: number | null; go: Go; seen?: number; versions: Version[] }) {
  const { it, go } = props
  const { model, scene, error } = useFacts(props.version)
  const refs = useRef<(HTMLElement | null)[]>([])
  const [at, setAt] = useState(0)
  const [picked, setPicked] = useState(0)
  useEffect(() => {
    const on = () => {
      let i = 0
      refs.current.forEach((el, j) => {
        if (el && el.getBoundingClientRect().top < 160) i = j
      })
      setAt(i)
    }
    window.addEventListener('scroll', on, true)
    return () => window.removeEventListener('scroll', on, true)
  }, [])
  if (error) return <Failure error={error} />
  if (!model || !scene) return <Loading q="What is this?" what="its explanation" />

  const name = props.summary.name
  const features = it.features ?? []
  const scenarios = it.scenarios ?? []
  const review = it.review ?? []
  const shown = features.slice(0, 5)
  const f = shown[picked]
  const boxes = (scene.model.nodes ?? []).filter((n) => ['application', 'datastore', 'queue'].includes(n.kind))
  const fLit = f ? boxesOfAll(featureFacts(f, model), model) : []
  const used = boxes.filter((b) => fLit.includes(b.id)).length
  const busiest = boxes
    .filter((b) => b.kind === 'application')
    .map((b) => ({ b, n: features.filter((x) => boxesOfAll(featureFacts(x, model), model).includes(b.id)).length }))
    .sort((a, b) => b.n - a.n)[0]?.b
  const parts = [
    ...(features.length ? [1] : []),
    ...(scenarios.length ? [2] : []),
    ...(review.length ? [3] : []),
  ]
  const order = [0, ...parts]

  return (
    <div className="reading tour">
      <h1 className="visually-hidden">{name}</h1>
      {props.seen !== undefined && props.version === null && <ReturnVisit seen={props.seen} versions={props.versions} go={go} />}
      <nav className="tour-strip" aria-label="The tour">
        <span className="tour-strip-name">The tour</span>
        {order.map((p, i) => (
          <button
            key={p}
            className={`tour-step ${i === at ? 'on' : ''} ${i < at ? 'done' : ''}`}
            aria-current={i === at ? 'step' : undefined}
            onClick={() => refs.current[i]?.scrollIntoView({ behavior: 'smooth', block: 'start' })}
          >
            <span className="tour-n">{i + 1}</span> {tourParts[p]}
          </button>
        ))}
        <span className="tour-time muted">about three minutes</span>
      </nav>

      <section ref={(el) => void (refs.current[0] = el)} className="tour-part">
        <div className="page-q">1 · What is it?</div>
        <p className="tour-thesis interp-sentence">{it.thesis.text}</p>
        <Cites ids={it.thesis.cites ?? []} model={model} it={it} go={go} />
        <div className="tour-mark">
          <AIMark line="Written by AI from the facts · every sentence opens onto what it rests on" />
          <button className="link-btn" onClick={() => go({ screen: 'written' })}>
            How much can you trust it? →
          </button>
        </div>
        {busiest && (
          <p className="tour-ask">
            Or ask it something:{' '}
            <button className="link-btn" onClick={() => go({ screen: 'part', focus: busiest.id, question: 'down' })}>
              What happens if {busiest.name} is down?
            </button>
          </p>
        )}
      </section>

      {features.length > 0 && (
        <section ref={(el) => void (refs.current[order.indexOf(1)] = el)} className="tour-part">
          <div className="page-q">{order.indexOf(1) + 1} · What is it for?</div>
          <p className="lede">Pick a feature to see which parts of {name} it uses.</p>
          <div className="tour-features">
            <ul className="tour-feature-list">
              {shown.map((x, i) => (
                <li key={x.name + i}>
                  <button className={`tour-feature ${i === picked ? 'on' : ''}`} aria-pressed={i === picked} onClick={() => setPicked(i)}>
                    <span className="tour-feature-name">{x.name}</span>
                    <span className="tour-feature-says">{x.says}</span>
                  </button>
                </li>
              ))}
              <li>
                <button className="link-btn" onClick={() => go({ screen: 'features' })}>
                  All {features.length} features →
                </button>
              </li>
            </ul>
            <div className="tour-map">
              <ViewDiagram scene={scene} highlight={{ lit: new Set(fLit) }} find="" onFind={() => {}} small />
              {f && (
                <p className="small">
                  <em>{f.name}</em> uses {used} of the {boxes.length} parts.{' '}
                  <button className="link-btn" onClick={() => go({ screen: 'feature', focus: featureSlugs(it)[picked] })}>
                    Open the feature →
                  </button>
                </p>
              )}
            </div>
          </div>
        </section>
      )}

      {scenarios.length > 0 && (
        <section ref={(el) => void (refs.current[order.indexOf(2)] = el)} className="tour-part">
          <div className="page-q">{order.indexOf(2) + 1} · What happens when…?</div>
          <p className="lede">
            {scenarios.length === 1 ? 'One story explains' : `${scenarios.length} stories explain`} how {name} works. Play one and watch it cross the parts, step by step.
          </p>
          <ScenarioCards it={it} model={model} go={go} />
        </section>
      )}

      {review.length > 0 && (
        <section ref={(el) => void (refs.current[order.indexOf(3)] = el)} className="tour-part">
          <div className="page-q">{order.indexOf(3) + 1} · What should you know?</div>
          <p className="lede">What an architect would point out, reading the same facts.</p>
          <Points points={review.slice(0, 3)} model={model} it={it} go={go} />
          {review.length > 3 && (
            <button className="link-btn" onClick={() => go({ screen: 'review' })}>
              All {review.length} points →
            </button>
          )}
        </section>
      )}

      <section className="tour-close">
        <p>
          That is {name}, in {order.length} {order.length === 1 ? 'part' : 'parts'}. The next time you open it, this page starts with what changed since today.
        </p>
        <div className="tour-doors">
          <button className="link-btn" onClick={() => go({ screen: 'architecture', level: 'container' })}>
            See the architecture, as C4 →
          </button>
          <button className="link-btn" onClick={() => go({ screen: 'reference' })}>
            Every part, in Reference →
          </button>
          <button className="link-btn" onClick={() => go({ screen: 'about' })}>
            What archdoc couldn't read →
          </button>
        </div>
      </section>
    </div>
  )
}

/** Review points, each tagged Observation or Question, with what it rests on. */
function Points({ points, model, it, go }: { points: NonNullable<Interpretation['review']>; model: Model; it: Interpretation; go: Go }) {
  return (
    <ol className="points">
      {points.map((p, i) => (
        <li key={i} className="point">
          <span className={`point-kind ${p.kind}`}>{p.kind === 'question' ? 'Question' : 'Observation'}</span>
          <span className="interp-sentence">{p.text}</span> <Cites ids={p.cites} model={model} it={it} go={go} />
        </li>
      ))}
    </ol>
  )
}

// ——— The review ———

/** Home › What an architect would point out (app-design.md §10, point 10): the whole review. */
export function Review(props: { it?: Interpretation; missing?: string; version: number | null; go: Go }) {
  const { model, error } = useFacts(props.version)
  if (!props.it) return <ExplainedState q="What should you know?" missing={props.missing} go={props.go} />
  if (error) return <Failure error={error} />
  if (!model) return <Loading q="What should you know?" what="the review" />
  const review = props.it.review ?? []
  const questions = review.filter((p) => p.kind === 'question').length
  return (
    <div className="reading">
      <div className="page-q">What should you know?</div>
      <h1 className="page-title">What an architect would point out</h1>
      <p className="lede">
        {review.length} {review.length === 1 ? 'point' : 'points'}, reading the same facts: {review.length - questions} the facts show, {questions} they cannot settle.
      </p>
      <AIMark line="Written by AI from the facts · each point names what it rests on" />
      <Points points={review} model={model} it={props.it} go={props.go} />
    </div>
  )
}

/** A view that needs the explanation, where there is none — or while it is read. */
function ExplainedState({ q: question, missing, go }: { q: string; missing?: string; go: Go }) {
  if (missing === undefined) return <Loading q={question} what="its explanation" />
  return (
    <div className="reading">
      <div className="page-q">{question}</div>
      <NotExplained why={missing} go={go} />
    </div>
  )
}

// ——— Features ———

/** Features by purpose (app-design-screens §8): what the program is for, each one a way onto its parts. */
export function FeatureList(props: { it: Interpretation; version: number | null; go: Go }) {
  const { it, go } = props
  const { model, error } = useFacts(props.version)
  if (error) return <Failure error={error} />
  if (!model) return <Loading q="What is it for?" what="its features" />
  const features = it.features ?? []
  const slugs = featureSlugs(it)
  const placed = new Set(features.flatMap((f) => f.members))
  const ways = (model.entries ?? []).length
  const left = (model.entries ?? []).filter((e) => !placed.has(e.id)).length
  return (
    <div className="reading">
      <div className="page-q">What is it for?</div>
      <h1 className="page-title">Features</h1>
      <p className="lede">{features.length} features, by what each one is for.</p>
      <AIMark line={`${features.length} features, named by AI · what each one contains is proven`} />
      <div className="feature-tiles">
        {features.map((f, i) => (
          <button key={slugs[i]} className="feature-tile" onClick={() => go({ screen: 'feature', focus: slugs[i] })}>
            <span className="feature-tile-name">{f.name}</span>
            <span className="feature-tile-says">{f.says}</span>
            <span className="muted small">{f.members.length} {f.members.length === 1 ? 'way in' : 'ways in'}</span>
          </button>
        ))}
      </div>
      {ways > 0 && (
        <p className="small muted">
          {left === 0 ? `Every page, route, command and job is in a feature.` : `Every page, route, command and job is in a feature but ${left}.`}{' '}
          <button className="link-btn" onClick={() => go({ screen: 'entries' })}>
            Every entry point →
          </button>
        </p>
      )}
    </div>
  )
}

/** One feature (app-design-screens §8): an archdoc view on Apps and services, its parts lit. */
export function OneFeature(props: { it?: Interpretation; missing?: string; route: Route; version: number | null; go: Go }) {
  const { it, go } = props
  const { model, scene, error } = useFacts(props.version)
  if (!it) return <ExplainedState q="What is it for?" missing={props.missing} go={go} />
  if (error) return <Failure error={error} />
  if (!model || !scene) return <Loading q="What is it for?" what="the feature" />
  const found = featureAt(it, props.route.focus ?? '')
  if (!found) {
    return (
      <div className="reading">
        <div className="page-q">What is it for?</div>
        <p className="lede">There’s no feature at this address in this version’s explanation.</p>
        <button className="link-btn" onClick={() => go({ screen: 'features' })}>
          Every feature →
        </button>
      </div>
    )
  }
  const { feature: f } = found
  const h = lit(featureFacts(f, model), model)
  const boxes = (scene.model.nodes ?? []).filter((n) => ['application', 'datastore', 'queue'].includes(n.kind))
  const used = boxes.filter((b) => h.lit.has(b.id)).length
  const scenario = (it.scenarios ?? []).findIndex((s) => f.members.includes(s.flow))
  const tables = (model.nodes ?? []).filter((n) => n.kind === 'table').length
  return (
    <div className="part-view">
      <div className="part-body">
        <div className="band">
          <div className="band-kind">Feature · an archdoc view, drawn on Apps and services</div>
          <h1 className="band-title">{f.name}</h1>
          <p className="band-line interp-sentence">{f.says}</p>
          <p className="band-meta">
            It uses {used} of the {boxes.length} parts. The others are dimmed. {f.members.length} {f.members.length === 1 ? 'way in' : 'ways in'}.{' '}
            {f.mark === 'written-by-ai' ? <AIMark line="Named by AI · what it contains is proven" /> : <span>Grouped by a rule</span>}
          </p>
        </div>
        <ViewDiagram scene={scene} highlight={h} find={props.route.q ?? ''} onFind={(v) => go({ screen: 'feature', focus: props.route.focus, q: v || undefined }, { replace: true })} />
        <section className="part-list" aria-label="What it holds">
          <div className="part-list-head">
            <h2>Its ways in</h2>
          </div>
          <ul className="explained-members">
            {f.members.slice(0, 40).map((m) => {
              const t = citeTarget(m, model, it)
              return t ? (
                <li key={m}>
                  <button className="part-link" onClick={() => go(t.to)}>
                    {t.label}
                  </button>
                </li>
              ) : null
            })}
            {f.members.length > 40 && <li className="muted small">and {f.members.length - 40} more, in Every entry point</li>}
          </ul>
        </section>
        <div className="step-away">
          <span className="page-q">One step away</span>
          {scenario >= 0 && (
            <button className="link-btn" onClick={() => go({ screen: 'scenario', focus: scenarioSlugs(it)[scenario], way: 'played' })}>
              Its scenario: {it.scenarios![scenario].title} →
            </button>
          )}
          <button className="link-btn" onClick={() => go({ screen: 'architecture', level: 'container' })}>
            The whole container diagram, in Architecture →
          </button>
          {tables > 0 && (
            <button className="link-btn" onClick={() => go({ screen: 'data' })}>
              What it stores, in Data →
            </button>
          )}
        </div>
      </div>
    </div>
  )
}

// ——— Scenarios ———

function Dots({ s, at, onPick }: { s: Scenario; at?: number; onPick?: (i: number) => void }) {
  const first = s.steps.findIndex((x) => x.background)
  return (
    <span className="step-dots" aria-hidden={onPick ? undefined : true}>
      {s.steps.map((x, i) => (
        <span key={i} className="step-dot-wrap">
          {i === first && i > 0 && <span className="step-rule" />}
          {onPick ? (
            <button
              className={`step-dot ${x.background ? 'later' : ''} ${at === i ? 'now' : at !== undefined && i < at ? 'past' : ''}`}
              aria-label={`Step ${i + 1}`}
              aria-current={at === i ? 'step' : undefined}
              onClick={() => onPick(i)}
            />
          ) : (
            <span className={`step-dot ${x.background ? 'later' : ''}`} />
          )}
        </span>
      ))}
    </span>
  )
}

/** A card per scenario: its question, its steps, the parts it crosses, and three ways in. */
function ScenarioCards({ it, model, go, full }: { it: Interpretation; model: Model; go: Go; full?: boolean }) {
  const slugs = scenarioSlugs(it)
  const name = (id: string) => (model.nodes ?? []).find((n) => n.id === id)?.name ?? id
  return (
    <div className="scenario-cards">
      {(it.scenarios ?? []).map((s, i) => (
        <div key={slugs[i]} className="scenario-card">
          <div className="scenario-card-title">{s.title}</div>
          <Dots s={s} />
          <div className="muted small">{stepsLine(s)}</div>
          {full && (
            <div className="scenario-crosses">
              {boxesOfAll(s.steps.flatMap((x) => x.crosses ?? []), model).map((b) => (
                <span key={b} className="tag">
                  {name(b)}
                </span>
              ))}
            </div>
          )}
          <div className="scenario-ways">
            <button className="btn btn-ink" onClick={() => go({ screen: 'scenario', focus: slugs[i], way: 'played' })}>
              Play it
            </button>
            {full && (
              <>
                <button className="link-btn" onClick={() => go({ screen: 'scenario', focus: slugs[i], way: 'seq' })}>
                  Sequence
                </button>
                <button className="link-btn" onClick={() => go({ screen: 'scenario', focus: slugs[i], way: 'steps' })}>
                  Steps
                </button>
              </>
            )}
          </div>
        </div>
      ))}
    </div>
  )
}

/** The scenario list (app-design-screens §9). */
export function Scenarios(props: { it?: Interpretation; missing?: string; summary: Summary; version: number | null; go: Go }) {
  const { model, error } = useFacts(props.version)
  if (!props.it) return <ExplainedState q="What happens when someone does the things that matter?" missing={props.missing} go={props.go} />
  if (error) return <Failure error={error} />
  if (!model) return <Loading q="What happens when someone does the things that matter?" what="its scenarios" />
  const n = (props.it.scenarios ?? []).length
  return (
    <div className="reading">
      <div className="page-q">What happens when someone does the things that matter?</div>
      <h1 className="page-title">Scenarios</h1>
      <p className="lede">
        {n === 0
          ? `The explanation tells no story of ${props.summary.name}: none of its flows was long enough to tell.`
          : `${n === 1 ? 'One story explains' : `${n} stories explain`} how ${props.summary.name} works, told across the steps archdoc followed in the code.`}
      </p>
      {n > 0 && <AIMark line="Told by AI · each step stands for proven steps of its flows" />}
      <ScenarioCards it={props.it} model={model} go={props.go} full />
    </div>
  )
}

const wayNames: Record<'played' | 'seq' | 'steps', string> = { played: 'Played on the parts', seq: 'As a sequence', steps: 'As steps' }

/** One scenario, read three ways on one player (app-design-screens §9). */
export function OneScenario(props: { it?: Interpretation; missing?: string; route: Route; version: number | null; go: Go }) {
  const { it, go, route } = props
  const { model, scene, error } = useFacts(props.version)
  const found = it ? scenarioAt(it, route.focus ?? '') : undefined
  const s = found?.scenario
  const total = s?.steps.length ?? 0
  const at = Math.min(Math.max((route.step ?? 1) - 1, 0), Math.max(total - 1, 0))
  const way = route.way ?? 'played'
  const [playing, setPlaying] = useState(false)
  const move = (patch: Partial<Route>, replace = true) => go({ screen: 'scenario', focus: route.focus, way, step: at + 1, ...patch }, { replace })
  const toStep = (i: number) => move({ step: i + 1 })

  // Playing moves a step about every two and a half seconds and stops at the last.
  useEffect(() => {
    if (!playing || !s) return
    if (at >= total - 1) {
      setPlaying(false)
      return
    }
    const t = setTimeout(() => toStep(at + 1), 2500)
    return () => clearTimeout(t)
  })
  // Space plays, ← → step.
  useEffect(() => {
    if (!s) return
    const on = (e: KeyboardEvent) => {
      // Typing is not playing: a key in a field is the field's. The target may be the window itself.
      const t = e.target
      if (t instanceof Element && t.closest('input, select, textarea, [contenteditable]')) return
      if (e.key === ' ') {
        e.preventDefault()
        if (at >= total - 1) toStep(0)
        setPlaying((p) => !p || at >= total - 1)
      } else if (e.key === 'ArrowRight' && at < total - 1) toStep(at + 1)
      else if (e.key === 'ArrowLeft' && at > 0) toStep(at - 1)
    }
    window.addEventListener('keydown', on)
    return () => window.removeEventListener('keydown', on)
  })

  if (!it) return <ExplainedState q="What happens when…?" missing={props.missing} go={go} />
  if (error) return <Failure error={error} />
  if (!model || !scene) return <Loading q="What happens when…?" what="the scenario" />
  if (!s) {
    return (
      <div className="reading">
        <div className="page-q">What happens when…?</div>
        <p className="lede">There’s no scenario at this address in this version’s explanation.</p>
        <button className="link-btn" onClick={() => go({ screen: 'scenarios' })}>
          Every scenario →
        </button>
      </div>
    )
  }
  const { wait, later } = stepSplit(s)
  const step = s.steps[at]
  const feature = (it.features ?? []).findIndex((f) => f.members.includes(s.flow))
  const end = at >= total - 1

  return (
    <div className="part-view scenario-view">
      <div className="explorer-toolbar part-bar">
        <div className="map-group" role="tablist" aria-label="How it is read">
          {(['played', 'seq', 'steps'] as const).map((w) => (
            <button key={w} role="tab" aria-selected={way === w} className={`lens-btn ${way === w ? 'on' : ''}`} onClick={() => move({ way: w }, false)}>
              {wayNames[w]}
            </button>
          ))}
        </div>
      </div>
      <div className="part-body">
        <div className="band">
          <div className="band-kind">Scenario · written by AI</div>
          <h1 className="band-title">{s.title}</h1>
          <p className="band-meta">
            {total} {total === 1 ? 'step' : 'steps'}
            {later ? `: ${wait} while you wait, ${later} in the background` : ''}. Each one stands for proven steps of its flows.
          </p>
        </div>

        <div className="player" role="group" aria-label="The player">
          <button
            className="btn btn-ink"
            onClick={() => {
              if (end) toStep(0)
              setPlaying(end ? true : !playing)
            }}
          >
            {playing ? 'Pause' : end ? 'Replay' : 'Play'}
          </button>
          <button className="btn" disabled={at === 0} onClick={() => toStep(at - 1)} aria-label="Previous step">
            ‹
          </button>
          <button className="btn" disabled={end} onClick={() => toStep(at + 1)} aria-label="Next step">
            ›
          </button>
          <Dots s={s} at={at} onPick={toStep} />
          <span className={`step-chip ${step.background ? 'later' : ''}`}>{step.background ? 'in the background' : 'while you wait'}</span>
          <span className="muted small">space plays · ← → step</span>
        </div>
        {way !== 'steps' && (
          <p className="player-now">
            <span className="mono">
              {at + 1} of {total}
            </span>{' '}
            <span className="interp-sentence">{step.text}</span>
          </p>
        )}

        {way === 'played' && <Played s={s} at={at} model={model} scene={scene} />}
        {way === 'seq' && <AsSequence s={s} at={at} model={model} title={`Sequence for ${s.title}`} />}
        {way === 'steps' && <AsSteps s={s} at={at} onPick={toStep} />}

        <div className="step-away">
          <span className="page-q">One step away</span>
          {feature >= 0 && (
            <button className="link-btn" onClick={() => go({ screen: 'feature', focus: featureSlugs(it)[feature] })}>
              Its feature: {it.features[feature].name} →
            </button>
          )}
          <button className="link-btn" onClick={() => go({ screen: 'architecture', level: `dynamic:${route.focus}` })}>
            The same story as a C4 dynamic diagram, in Architecture →
          </button>
          <button className="link-btn" onClick={() => go({ screen: 'entries', focus: s.flow })}>
            The flow it follows, step by step →
          </button>
        </div>
      </div>
    </div>
  )
}

/** Played on the parts: what the story has reached lit, the current step's parts tagged. */
function Played({ s, at, model, scene }: { s: Scenario; at: number; model: Model; scene: SceneResponse }) {
  const { reached, current } = scenarioReach(s, at, model)
  const h: Highlight = { lit: new Set(reached), tags: new Map(current.map((b) => [b, `step ${at + 1}`])), tone: 'interp' }
  return <ViewDiagram scene={scene} highlight={h} subject={current[0]} find="" onFind={() => {}} />
}

/** As steps: the sentences, under While you wait and In the background, each with what it stands for. */
function AsSteps({ s, at, onPick }: { s: Scenario; at: number; onPick: (i: number) => void }) {
  const group = (later: boolean) => s.steps.map((x, i) => ({ x, i })).filter(({ x }) => !!x.background === later)
  return (
    <div className="as-steps">
      {[false, true].map((later) => {
        const g = group(later)
        if (!g.length) return null
        return (
          <section key={String(later)}>
            <h2 className="as-steps-head">{later ? 'In the background' : 'While you wait'}</h2>
            <ol className="as-steps-list">
              {g.map(({ x, i }) => (
                <li key={i} className={i === at ? 'now' : ''}>
                  <button className="as-step" onClick={() => onPick(i)} aria-current={i === at ? 'step' : undefined}>
                    <span className="mono as-step-n">{i + 1}</span>
                    <span className="interp-sentence">{x.text}</span>
                    <span className="muted small">
                      {x.background ? 'stands for queued jobs’ flows' : x.from === x.to ? `stands for step ${x.from} of its flow` : `stands for steps ${x.from}–${x.to} of its flow`}
                    </span>
                  </button>
                </li>
              ))}
            </ol>
          </section>
        )
      })}
    </div>
  )
}

/**
 * As a sequence: a lifeline for each part the story crosses, in the order it meets them, and a
 * message for each step — from the part the story was in to the one it reaches, numbered; a step
 * that stays in one part is a note on its lifeline. Background steps sit in a grey group, dashed.
 */
export function AsSequence({ s, at, model, title }: { s: Scenario; at?: number; model: Model; title: string }) {
  const nodes = new Map((model.nodes ?? []).map((n) => [n.id, n]))
  // A story that stays in one part is drawn on the parts inside it: its components and tables.
  const outer = s.steps.map((x) => boxesOfAll(x.crosses ?? [], model))
  const inner = s.steps.map((x) => {
    const out: string[] = []
    for (const id of x.crosses ?? []) {
      const e = (model.entries ?? []).find((y) => y.id === id)
      const lane = e ? e.component ?? e.container : nodes.has(id) ? id : undefined
      if (lane && !out.includes(lane)) out.push(lane)
    }
    return out
  })
  const distinct = (per: string[][]) => [...new Set(per.flat())]
  const inside = distinct(outer).length <= 1 && distinct(inner).length > 1
  const per = inside ? inner : outer
  const lanes = distinct(per)
  if (!lanes.length) return <p className="muted">The steps of this scenario cross no part archdoc draws.</p>
  const name = (id: string) => {
    const n = nodes.get(id)
    return n ? (n.kind === 'table' ? `table ${n.name}` : n.name) : id.replace(/^[a-z]+:/, '')
  }
  const COL = 150
  const HEAD = 48
  const ROW = 40
  const PAD = 20
  const x = (b: string) => PAD + lanes.indexOf(b) * COL + COL / 2
  // Each step's message goes from the part whose code runs it — the first it crosses — to the first
  // other part it reaches; a step that stays in one part is a note on its lifeline.
  let last = lanes[0]
  const msgs = s.steps.map((st, i) => {
    const boxes = per[i]
    const from = boxes[0] ?? last
    const to = boxes.find((b) => b !== from) ?? from
    last = from
    return { i, from, to, later: !!st.background, words: `${i + 1} ${st.text.split(/[,.;:]/)[0].split(' ').slice(0, 6).join(' ')}` }
  })
  // A note is as wide as its words; the picture is as wide as its widest note.
  const noteW = (words: string) => Math.max(60, Math.round(words.length * 6.2) + 14)
  const width = Math.max(PAD * 2 + lanes.length * COL, ...msgs.filter((m) => m.from === m.to).map((m) => x(m.from) + 6 + noteW(m.words) + PAD))
  const height = HEAD + 20 + msgs.length * ROW + 12
  const firstLater = msgs.findIndex((m) => m.later)
  const container = inside ? name(distinct(outer)[0] ?? '') : ''
  return (
    <figure className="sequence scenario-sequence">
      <figcaption className="c4-head-title">
        <span className="c4-title">{title}</span>
        <span className="c4-plain">
          {inside ? `Inside ${container}: a box per component or table it crosses` : 'A box per part'} · a solid arrow, while you wait · dashed, in the background
        </span>
      </figcaption>
      <svg width={width} height={height} role="img" aria-label={title}>
        {firstLater >= 0 && <rect x={4} y={HEAD + 8 + firstLater * ROW} width={width - 8} height={(msgs.length - firstLater) * ROW + 4} className="seq-later" />}
        {firstLater >= 0 && (
          <text x={10} y={HEAD + 22 + firstLater * ROW} className="seq-later-label">
            In the background
          </text>
        )}
        {lanes.map((b) => (
          <g key={b}>
            <rect x={x(b) - COL / 2 + 8} y={6} width={COL - 16} height={30} rx={3} className="seq-head" />
            <text x={x(b)} y={25} textAnchor="middle" className="seq-head-text">
              {name(b).length > 20 ? name(b).slice(0, 19) + '…' : name(b)}
            </text>
            <line x1={x(b)} x2={x(b)} y1={36} y2={height - 4} className="seq-life" />
          </g>
        ))}
        {msgs.map((m) => {
          const y = HEAD + 24 + m.i * ROW
          const cls = `seq-msg ${m.later ? 'later' : ''} ${at === m.i ? 'now' : at !== undefined && m.i > at ? 'to-come' : ''}`
          if (m.from === m.to) {
            return (
              <g key={m.i} className={cls}>
                <rect x={x(m.from) + 6} y={y - 12} width={noteW(m.words)} height={18} rx={2} className="seq-note" />
                <text x={x(m.from) + 12} y={y + 1} className="seq-text">
                  {m.words}
                </text>
              </g>
            )
          }
          const x1 = x(m.from)
          const x2 = x(m.to)
          const dir = x2 > x1 ? 1 : -1
          return (
            <g key={m.i} className={cls}>
              <line x1={x1} x2={x2 - dir * 6} y1={y} y2={y} className="seq-line" />
              <path d={`M${x2} ${y} L${x2 - dir * 7} ${y - 4} L${x2 - dir * 7} ${y + 4}z`} className="seq-arrow" />
              <text x={(x1 + x2) / 2} y={y - 6} textAnchor="middle" className="seq-text">
                {m.words}
              </text>
            </g>
          )
        })}
      </svg>
    </figure>
  )
}

// ——— Architecture: the C4 dynamic diagram ———

/**
 * The dynamic diagram (app-design-screens §4, a scenario): Apps and services holding only the parts a
 * scenario crosses, each tagged with the numbers of the steps that use it. Nothing dimmed, nothing
 * moves; a row of scenarios to pick from, and the way to play it.
 */
export function Dynamic(props: { it?: Interpretation; missing?: string; route: Route; summary: Summary; version: number | null; go: Go }) {
  const { it, go } = props
  const { model, scene, error } = useFacts(props.version)
  const at = (props.route.level ?? '').replace(/^dynamic:/, '')
  const slugs = it ? scenarioSlugs(it) : []
  const found = it ? scenarioAt(it, at) ?? (slugs[0] ? scenarioAt(it, slugs[0]) : undefined) : undefined
  const per = useMemo(() => (found && model ? found.scenario.steps.map((x) => boxesOfAll(x.crosses ?? [], model)) : []), [found, model])
  if (!it) return <ExplainedState q="How is it put together?" missing={props.missing} go={go} />
  if (error) return <Failure error={error} />
  if (!model || !scene) return <Loading what="the dynamic diagram" />
  if (!found) {
    return (
      <div className="reading">
        <div className="page-q">How is it put together?</div>
        <p className="lede">The explanation tells no scenario, so there is no dynamic diagram.</p>
      </div>
    )
  }
  const s = found.scenario
  const numbers = new Map<string, number[]>()
  per.forEach((boxes, i) => boxes.forEach((b) => numbers.set(b, [...(numbers.get(b) ?? []), i + 1])))
  // Nothing is dimmed and nothing moves (app-design-screens §4): every box stays, the ones the story
  // crosses carry the numbers of their steps.
  const every = new Set((scene.model.nodes ?? []).map((n) => n.id))
  const h: Highlight = { lit: every, tags: new Map([...numbers].map(([b, ns]) => [b, ns.join(', ')])), tone: 'ink' }
  const title = `Dynamic diagram for ${props.summary.name}: ${s.title.toLowerCase()}`
  return (
    <div className="part-view">
      <div className="explorer-toolbar part-bar">
        <div className="map-group" role="tablist" aria-label="The scenario">
          {(it.scenarios ?? []).map((x, i) => (
            <button key={slugs[i]} role="tab" aria-selected={slugs[i] === slugs[found.index]} className={`lens-btn ${slugs[i] === slugs[found.index] ? 'on' : ''}`} onClick={() => go({ screen: 'architecture', level: `dynamic:${slugs[i]}` })}>
              {x.title}
            </button>
          ))}
        </div>
        <button className="link-btn" onClick={() => go({ screen: 'scenario', focus: slugs[found.index], way: 'played' })}>
          Play it, in Scenarios →
        </button>
      </div>
      <div className="part-body">
        <div className="c4-head">
          <div className="c4-head-title">
            <span className="c4-title">{title}</span>
            <span className="c4-plain">A scenario, on Apps and services</span>
          </div>
          <p className="c4-shows">
            The {numbers.size} {numbers.size === 1 ? 'part' : 'parts'} the story crosses, each with the numbers of the steps that use it. The steps are in the list beneath.
          </p>
        </div>
        <ViewDiagram scene={scene} highlight={h} find={props.route.q ?? ''} onFind={(v) => go({ screen: 'architecture', level: props.route.level, q: v || undefined }, { replace: true })} />
        <ol className="dynamic-steps">
          {s.steps.map((x, i) => (
            <li key={i} className={x.background ? 'later' : ''}>
              <span className="interp-sentence">{x.text}</span>
            </li>
          ))}
        </ol>
        <AIMark line="The steps are written by AI · each stands for proven steps of its flows" />
      </div>
    </div>
  )
}

// ——— Under a diagram: what to notice, and areas ———

/** What to notice (app-design-screens §4, item 4): beneath a diagram, where the explanation wrote one. */
export function Notice({ it, view, model, go }: { it?: Interpretation; view?: string; model?: Model; go: Go }) {
  const note = (it?.notes ?? []).find((n) => n.diagram === view)
  if (!it || !note || !model) return null
  return (
    <aside className="notice-block" aria-label="What to notice">
      <span className="notice-diamond" aria-hidden="true">◆</span>
      <span className="page-q">What to notice</span>
      <span className="interp-sentence">{note.text}</span> <Cites ids={note.cites} model={model} it={it} go={go} />
    </aside>
  )
}

/** The areas of the divided container: its components by responsibility, the one used everywhere first. */
export function Areas({ it, container, model, go }: { it?: Interpretation; container?: string; model?: Model; go: Go }) {
  if (!it?.areas?.length || !container || it.areas_of !== container || !model) return null
  const name = (id: string) => (model.nodes ?? []).find((n) => n.id === id)?.name ?? id
  const areas = [...it.areas].sort((a, b) => Number(!!b.everywhere) - Number(!!a.everywhere))
  return (
    <details className="areas-block" aria-label="Its areas">
      <summary className="page-q">
        Its {areas.length} areas <AIMark line="grouped by AI from archdoc’s seeds · each component is proven" />
      </summary>
      <div className="area-tiles">
        {areas.map((a) => (
          <div key={a.name} className={`area-tile ${a.everywhere ? 'everywhere' : ''}`}>
            <div className="area-name">
              {a.name}
              {a.everywhere && <span className="tag">Used everywhere</span>}
            </div>
            <div className="small">{a.says}</div>
            <div className="area-members">
              {a.members.map((m) => (
                <button key={m} className="part-link small" onClick={() => go({ screen: 'component', focus: m })}>
                  {name(m)}
                </button>
              ))}
            </div>
          </div>
        ))}
      </div>
    </details>
  )
}
