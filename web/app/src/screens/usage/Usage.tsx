import { useMemo } from 'react'
import type { Entry, ModelResponse, SceneResponse, Summary } from '../../api'
import { useApi } from '../../api'
import type { Route } from '../../route'
import { Cite } from '../../ui/Cite'
import { Failure, Loading, TruthMark } from '../../ui/marks'
import { ViewDiagram } from '../explorer/ViewDiagram'
import type { Face } from './faces'
import { addressOf, countOf, faceHeading, faceMark, faceName, faceOfEntry, facesOf, matches, nameOf, setsOff, setsOffSentence, usedSentence } from './faces'

type Go = (r: Partial<Route>, o?: { keep?: boolean; replace?: boolean }) => void

/**
 * Usage (platform-surface §17.3; app-design-screens §10): how it is used. A tab for each face — the
 * screens of an app, the API of a service, the command line of a tool — on one layout: what can be
 * done at the left, the chosen entry at the right, with what handles it, what it uses, what it sets
 * off, and where it runs. Every value is the code's, cited; nothing here is written by AI.
 *
 * The complete list of entry points, background jobs included, stays in Reference.
 */
export function Usage(props: { route: Route; go: Go; version: number | null; summary: Summary }) {
  const { route, go } = props
  const v = props.version ? `?version=${props.version}` : ''
  const model = useApi<ModelResponse>(`/api/model${v}`)
  const scene = useApi<SceneResponse>(`/api/scene?view=${props.summary.tiny ? 'context' : 'container'}${props.version ? `&version=${props.version}` : ''}`)
  const m = model.data?.model
  const faces = useMemo(() => facesOf(m?.entries ?? []), [m])
  const nodes = useMemo(() => new Map((m?.nodes ?? []).map((n) => [n.id, n])), [m])
  const flows = useMemo(() => new Map((m?.flows ?? []).map((f) => [f.entry, f])), [m])
  const name = (id: string) => nodes.get(id)?.name ?? id.replace(/^[a-z]+:/, '')

  if (model.error) return <Failure error={model.error} />
  if (!m) return <Loading q="How is it used?" what="the ways in" />

  // The face named in the address; else the chosen entry's; else the first.
  const asked = route.focus ? m.entries?.find((e) => e.id === route.focus) : undefined
  const face = faces.find((f) => f.key === route.face) ?? faces.find((f) => asked && f.key === faceOfEntry(asked)) ?? faces[0]
  const open = (patch: Partial<Route>, replace = false) => go({ screen: 'usage', face: face?.key, q: route.q, ...patch }, { replace })

  return (
    <div className="reading wide usage">
      <div className="page-q">How is it used?</div>
      <h1 className="page-title">Usage</h1>
      <p className="usage-lede">{usedSentence(props.summary.name, faces, name)}</p>

      {!face ? (
        <p className="muted">
          <button className="link-btn" onClick={() => go({ screen: 'entries' })}>
            Every entry point, in Reference →
          </button>
        </p>
      ) : (
        <>
          <div className="usage-tabs" role="tablist" aria-label="How it is used">
            {faces.map((f) => (
              <button key={f.key} role="tab" aria-selected={f.key === face.key} className={`usage-tab ${f.key === face.key ? 'on' : ''}`} onClick={() => go({ screen: 'usage', face: f.key })}>
                <span className="usage-tab-name">{faceName[f.how]}</span>
                <span className="usage-tab-sub">
                  {name(f.container)} · {countOf(f)}
                </span>
              </button>
            ))}
          </div>
          <p className="usage-mark">
            <TruthMark state="proven" /> {faceMark(face, m.flows ?? [])}
          </p>
          <Columns face={face} route={route} open={open} go={go} name={name} scene={scene.data} flows={flows} />
        </>
      )}
    </div>
  )
}

function Columns(props: {
  face: Face
  route: Route
  open: (patch: Partial<Route>, replace?: boolean) => void
  go: Go
  name: (id: string) => string
  scene?: SceneResponse
  flows: Map<string, NonNullable<ModelResponse['model']['flows']>[number]>
}) {
  const { face, route } = props
  const shown = face.entries.filter((e) => matches(e, route.q ?? ''))
  const chosen = shown.find((e) => e.id === route.focus) ?? face.entries.find((e) => e.id === route.focus) ?? shown[0]

  return (
    <div className="usage-cols">
      <section className="usage-list" aria-label={faceHeading[face.how]}>
        <div className="usage-list-head">
          <h2>{faceHeading[face.how]}</h2>
          <input
            className="usage-filter"
            value={route.q ?? ''}
            placeholder="Filter by name, address or handler"
            aria-label="Filter"
            onChange={(e) => props.open({ q: e.target.value || undefined, focus: route.focus }, true)}
          />
        </div>
        <div className="usage-rows">
          {shown.map((e) => (
            <button key={e.id} className={`usage-row ${chosen?.id === e.id ? 'on' : ''}`} aria-pressed={chosen?.id === e.id} onClick={() => props.open({ focus: e.id })}>
              <span className="usage-row-name">{nameOf(e)}</span>
              {e.summary && <span className="usage-row-address">{addressOf(e)}</span>}
              <span className="usage-row-line">{e.handler}</span>
            </button>
          ))}
          {shown.length === 0 && <p className="muted usage-none">Nothing here matches “{route.q}”.</p>}
        </div>
        <div className="usage-list-foot">
          {shown.length === face.entries.length ? `${shown.length} of ${countOf(face)}` : `${shown.length} shown of ${countOf(face)}`} ·{' '}
          <button className="link-btn" onClick={() => props.go({ screen: 'entries' })}>
            Every entry point, in Reference →
          </button>
        </div>
      </section>
      {chosen && <Chosen e={chosen} name={props.name} scene={props.scene} flow={props.flows.get(chosen.id)} go={props.go} />}
    </div>
  )
}

/** The chosen entry: what it is, what handles it, what it uses, what it sets off, and where it runs. */
function Chosen(props: { e: Entry; name: (id: string) => string; scene?: SceneResponse; flow?: Parameters<typeof setsOff>[0]; go: Go }) {
  const { e } = props
  const lit = useMemo(() => ({ lit: new Set([e.container]) }), [e.container])
  return (
    <aside className="usage-entry" aria-label="The chosen entry">
      <h2 className="usage-entry-name">{nameOf(e)}</h2>
      {e.summary && <div className="usage-entry-address">{addressOf(e)}</div>}
      {e.summary ? (
        <p className="usage-entry-says">
          What its code says it does. <Cite p={e.summary_provenance} compact />
        </p>
      ) : (
        <p className="usage-entry-says muted">Its code states no summary, so it is named by its address.</p>
      )}

      <dl className="usage-facts">
        <dt>Handled by</dt>
        <dd>
          <span className="mono">{e.handler}</span> <Cite p={e.provenance} compact />
          {e.path_note && <div className="usage-note">{e.path_note}</div>}
        </dd>
        <dt>Uses</dt>
        <dd>
          {e.uses?.length ? (
            e.uses.map((u, i) => (
              <div key={i} className="usage-use">
                <span className="mono">{u.name}</span>
                {u.component && <span className="muted"> · in {props.name(u.component)}</span>} <Cite p={u.provenance} compact />
              </div>
            ))
          ) : (
            <span className="muted">Nothing it is given that archdoc could name.</span>
          )}
        </dd>
        <dt>Sets off</dt>
        <dd>
          {setsOffSentence(setsOff(props.flow))}
          {props.flow && props.flow.steps.length > 0 && (
            <div>
              <button className="link-btn" onClick={() => props.go({ screen: 'entries', focus: e.id })}>
                Its {props.flow.steps.length} steps, each at its line →
              </button>
            </div>
          )}
        </dd>
      </dl>

      <div className="usage-where">
        <div className="usage-where-head">Where it runs</div>
        {/* Keyed by its part, so a new part is framed anew. */}
        {props.scene && <ViewDiagram key={e.container} scene={props.scene} highlight={lit} find="" onFind={() => {}} small />}
        <p className="usage-where-line">
          It runs in {props.name(e.container)}. Which other parts each entry reaches isn’t recorded yet.
        </p>
      </div>
    </aside>
  )
}
