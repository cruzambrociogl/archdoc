import { useEffect, useMemo } from 'react'
import type { CoverageResponse, ModelResponse, SceneResponse, Session, Summary } from '../../api'
import { useApi } from '../../api'
import type { Route } from '../../route'
import { Cite } from '../../ui/Cite'
import { Failure, Loading, TruthMark } from '../../ui/marks'
import { ViewDiagram } from '../explorer/ViewDiagram'
import { aiHighlight, aiSentence, readHighlight, readSentence, unreadRows, unresolvedRows } from './facts'

type Go = (r: Partial<Route>, o?: { keep?: boolean; replace?: boolean }) => void

/** A list stops at five, with a way to see all (rule 15). */
const FIRST = 5

/**
 * What archdoc knows — the honesty map (platform-surface §17.3; app-design-screens §14). Two
 * archdoc views on Apps and services: the system drawn by how much of it archdoc read, and by which
 * words the AI wrote. Below, what couldn't be resolved and what wasn't read, each with its line.
 * The whole report — every count, the validator's findings, what reading cannot state — is one step
 * away, in About these docs.
 */
export function Trust(props: { route: Route; go: Go; version: number | null; summary: Summary; session?: Session; onNames?: (names: Record<string, string>) => void }) {
  const { route, go } = props
  const ai = route.screen === 'written'
  const q = props.version ? `&version=${props.version}` : ''
  const coverage = useApi<CoverageResponse>('/api/coverage')
  // Apps and services as Architecture shows it; a project too small for it is drawn in its context.
  const scene = useApi<SceneResponse>(`/api/scene?view=${props.summary.tiny ? 'context' : 'container'}${q}`)
  // The explanations the AI wrote live in the whole model.
  const model = useApi<ModelResponse>(ai ? `/api/model${props.version ? `?version=${props.version}` : ''}` : null)

  const nodes = useMemo(() => new Map((scene.data?.model.nodes ?? []).map((n) => [n.id, n])), [scene.data])
  const name = (id: string) => nodes.get(id)?.name ?? id.replace(/^[a-z]+:/, '')
  const onNames = props.onNames
  useEffect(() => {
    if (!scene.data || !onNames) return
    const names: Record<string, string> = {}
    for (const n of scene.data.model.nodes ?? []) names[n.id] = n.name
    onNames(names)
  }, [scene.data, onNames])

  const error = coverage.error ?? scene.error ?? model.error
  if (error) return <Failure error={error} />
  if (!coverage.data || !scene.data || (ai && !model.data)) return <Loading what="what archdoc knows" />

  const c = coverage.data
  const drawn = { ...scene.data.model, explanations: model.data?.model.explanations }
  const ids = (scene.data.model.nodes ?? []).map((n) => n.id)
  const move = (patch: Partial<Route>, replace = false) => go({ screen: route.screen, q: route.q, all: route.all, ...patch }, { replace })
  const commit = props.session?.commit ?? props.summary.commit

  return (
    <div className="part-view">
      <div className="explorer-toolbar part-bar">
        <div className="map-group" role="tablist" aria-label="What archdoc knows">
          <button role="tab" aria-selected={!ai} className={`lens-btn ${!ai ? 'on' : ''}`} onClick={() => go({ screen: 'about' })}>
            What was read
          </button>
          <button role="tab" aria-selected={ai} className={`lens-btn ${ai ? 'on' : ''}`} onClick={() => go({ screen: 'written' })}>
            What the AI wrote
          </button>
        </div>
      </div>

      <div className="part-body">
        <div className="band">
          <div className="band-kind">What archdoc knows · an archdoc view</div>
          <h1 className="band-title">{ai ? 'Which words did the AI write?' : 'How much of this picture can you trust?'}</h1>
          <p className="band-line">{ai ? aiSentence(drawn) : readSentence(c, name)}</p>
          <p className="band-meta">
            Version {props.session?.version ?? props.summary.latest_version}
            {commit ? ` from commit ${commit.slice(0, 7)}` : ''}
            {c.source ? `, read from ${c.source}` : ''}.
          </p>
        </div>

        {ai ? <AIKey /> : <ReadKey />}

        <ViewDiagram scene={scene.data} find={route.q ?? ''} highlight={ai ? aiHighlight(drawn) : readHighlight(c, ids)} onFind={(v) => move({ q: v || undefined }, true)} />

        {!ai && <Lists c={c} name={name} all={!!route.all} onAll={() => move({ all: route.all ? undefined : '1' }, true)} />}

        <button className="part-more" onClick={() => go({ screen: 'report' })}>
          <span>
            <span className="part-more-title">About these docs</span>
            <span className="part-more-sub">Every count, what the checks found, what reading the code cannot tell, and what left this machine</span>
          </span>
          <span aria-hidden="true">→</span>
        </button>
      </div>
    </div>
  )
}

/** The key of What was read: every style its diagram uses (rule 23). */
function ReadKey() {
  return (
    <div className="view-key" aria-label="Key">
      <span className="view-key-label">Key</span>
      <span className="view-key-item">
        <span className="key-box" /> Read in full
      </span>
      <span className="view-key-item">
        <span className="key-box hatch" /> Partly read, or not read
      </span>
      <span className="view-key-item">
        <span className="key-count">? 3</span> Calls to an address set at run time
      </span>
    </div>
  )
}

/** The key of What the AI wrote. */
function AIKey() {
  return (
    <div className="view-key" aria-label="Key">
      <span className="view-key-label">Key</span>
      <span className="view-key-item">
        <span className="key-box" /> From the code, its configuration or the catalog
      </span>
      <span className="view-key-item">
        <span className="key-tag">Written by AI</span> A box whose words the AI wrote
      </span>
    </div>
  )
}

/** What couldn't be resolved, and what wasn't read: each a list of five, and all on asking. */
function Lists(props: { c: CoverageResponse; name: (id: string) => string; all: boolean; onAll: () => void }) {
  const loose = unresolvedRows(props.c)
  const unread = unreadRows(props.c)
  const shown = props.all ? loose : loose.slice(0, FIRST)
  const shownUnread = props.all ? unread : unread.slice(0, FIRST)
  const more = loose.length > FIRST || unread.length > FIRST
  return (
    <>
      <section className="part-list" aria-label="What couldn’t be resolved">
        <div className="part-list-head">
          <h2>What couldn’t be resolved</h2>
          <span className="muted small">{loose.length}</span>
        </div>
        {loose.length === 0 && <p className="muted">Nothing: every call and import archdoc read went somewhere it could name.</p>}
        {shown.map((u, i) => (
          <div key={i} className="part-row">
            <div className="part-row-main">
              <span className="part-row-name">{props.name(u.box)}</span>
              <span className="part-tag unresolved">
                <TruthMark state="unresolved" /> couldn’t be resolved
              </span>
            </div>
            <div className="part-row-what mono">{u.what}</div>
            <div className="part-row-how">{u.why.charAt(0).toUpperCase() + u.why.slice(1)}.</div>
            <div className="part-row-cites">
              <Cite p={u.cite} compact />
            </div>
          </div>
        ))}
      </section>

      <section className="part-list" aria-label="What wasn’t read">
        <div className="part-list-head">
          <h2>What wasn’t read</h2>
          <span className="muted small">{unread.length}</span>
        </div>
        {unread.length === 0 && <p className="muted">Nothing: archdoc read every file it chose, all the way through.</p>}
        {shownUnread.map((u, i) => (
          <div key={i} className="part-row">
            <div className="part-row-main">
              {u.box && <span className="part-row-name">{props.name(u.box)}</span>}
              <span className={u.box ? 'part-row-what' : 'part-row-what mono'}>{u.what}</span>
              <span className="part-tag indirect">{u.kind}</span>
            </div>
            <div className="part-row-how">{u.why}</div>
            {u.files && (
              <div className="part-row-files mono">
                {(props.all ? u.files : u.files.slice(0, FIRST)).join(' · ')}
                {!props.all && u.files.length > FIRST ? ` · and ${u.files.length - FIRST} more` : ''}
              </div>
            )}
          </div>
        ))}
      </section>

      {more && (
        <button className="link-btn part-all" onClick={props.onAll}>
          {props.all ? 'Show the first five of each' : 'Show all of both lists'}
        </button>
      )}
    </>
  )
}
