import { useMemo } from 'react'
import type { DiffResponse, ModelResponse, SceneResponse, Summary, Version } from '../../api'
import { useApi } from '../../api'
import type { Route } from '../../route'
import { join, landed, tell } from '../../story'
import { Cite } from '../../ui/Cite'
import { Failure, Loading } from '../../ui/marks'
import { ViewDiagram } from '../explorer/ViewDiagram'

type Go = (r: Partial<Route>, o?: { keep?: boolean; replace?: boolean }) => void

const day = (iso?: string) => (iso ? new Date(iso).toLocaleDateString(undefined, { day: 'numeric', month: 'short' }) : '')

/**
 * One change, where it landed (platform-surface §17.3; app-design-screens §11): an archdoc view on
 * Apps and services as the version that made it draws it — the parts it touched lit, a tag on each
 * saying what happened to it, the rest dimmed. The change is one line of the version's story, told
 * from the engine's comparison, so it is proven; the line that shows it is cited.
 */
export function OneChange(props: { route: Route; go: Go; versions: Version[]; summary: Summary }) {
  const { route, go } = props
  const sorted = useMemo(() => [...props.versions].sort((a, b) => b.id - a.id), [props.versions])
  const to = route.v ?? sorted[0]?.id
  // Compared with the version before it, unless the link names another — Home's "since you last looked".
  const base = route.from ?? sorted.find((v) => to !== undefined && v.id < to)?.id
  const ready = to !== undefined && base !== undefined
  const diff = useApi<DiffResponse>(ready ? `/api/diff?from=${base}&to=${to}` : null)
  const before = useApi<ModelResponse>(ready ? `/api/model?version=${base}` : null)
  const after = useApi<ModelResponse>(ready ? `/api/model?version=${to}` : null)
  const scene = useApi<SceneResponse>(ready ? `/api/scene?view=${props.summary.tiny ? 'context' : 'container'}&version=${to}` : null)

  const error = diff.error ?? before.error ?? after.error ?? scene.error
  if (!ready) return <Gone go={go} why="There is no earlier version to compare this one with, so it has no changes to show." />
  if (error) return <Failure error={error} />
  if (!diff.data || !before.data || !after.data || !scene.data) return <Loading what="the change" />

  const { groups } = tell(diff.data, before.data, after.data)
  const group = groups.find((g) => g.items.some((it) => it.key === route.focus))
  const item = group?.items.find((it) => it.key === route.focus)
  if (!group || !item) return <Gone go={go} why={`Version ${to} has no change called “${route.focus}”.`} />

  const nodes = new Map([...(before.data.model.nodes ?? []), ...(after.data.model.nodes ?? [])].map((n) => [n.id, n]))
  const drawn = new Set((scene.data.model.nodes ?? []).map((n) => n.id))
  const where = landed(item.on ?? [], (id) => nodes.get(id)?.parent, drawn)
  const name = (id: string) => nodes.get(id)?.name ?? id
  const version = sorted.find((v) => v.id === to)
  const latest = to === sorted[0]?.id

  return (
    <div className="part-view">
      <div className="part-body">
        <div className="band">
          <div className="band-kind">
            Change · {day(version?.created_at)} · {group.title} · an archdoc view
          </div>
          <h1 className="band-title">{item.text}</h1>
          <p className="band-line">
            {where.lit.size
              ? `It landed in ${join([...where.lit].map(name).sort((a, b) => a.localeCompare(b)))}.`
              : 'What it touched is no longer drawn on this diagram.'}
          </p>
          <p className="band-meta">
            Version {to}
            {version?.commit ? ` · ${version.commit.slice(0, 7)}` : ''}, compared with version {base}. From the engine’s comparison, not written by AI.{' '}
            {item.cite?.file && <Cite p={item.cite} compact />}
          </p>
        </div>

        <div className="view-key" aria-label="Key">
          <span className="view-key-label">Key</span>
          <span className="view-key-item">
            <span className="key-box" /> What the change touched
          </span>
          <span className="view-key-item">
            <span className="key-tag ink">New route</span> What happened to it
          </span>
          <span className="view-key-item">
            <span className="key-box dim" /> Everything else
          </span>
        </div>

        <ViewDiagram scene={scene.data} highlight={{ lit: where.lit, tags: where.tags }} find={route.q ?? ''} onFind={(q) => go({ screen: 'change', v: route.v, focus: route.focus, from: route.from, q: q || undefined }, { replace: true })} />

        <div className="part-steps">
          <div className="part-steps-head">One step away</div>
          <button className="part-more" onClick={() => go({ screen: 'changes', v: latest ? undefined : to })}>
            <span>
              <span className="part-more-title">Every change in this version</span>
              <span className="part-more-sub">The version told whole, grouped by what each change means</span>
            </span>
            <span aria-hidden="true">→</span>
          </button>
          <button className="part-more" onClick={() => go({ screen: 'architecture', level: 'container', v: latest ? undefined : to })}>
            <span>
              <span className="part-more-title">Apps and services as it is {latest ? 'now' : `in version ${to}`}</span>
              <span className="part-more-sub">The C4 diagram, in Architecture</span>
            </span>
            <span aria-hidden="true">→</span>
          </button>
        </div>
      </div>
    </div>
  )
}

function Gone(props: { go: Go; why: string }) {
  return (
    <div className="explorer-state reading">
      <div className="page-q">One change</div>
      <p className="lede">{props.why}</p>
      <button className="link-btn" onClick={() => props.go({ screen: 'changes' })}>
        Changes →
      </button>
    </div>
  )
}
