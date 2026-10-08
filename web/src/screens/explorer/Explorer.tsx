import { useEffect, useMemo, useRef, useState } from 'react'
import { Background, BackgroundVariant, MiniMap, ReactFlow, ReactFlowProvider, ViewportPortal, useReactFlow, useStore } from '@xyflow/react'
import type { Node as FlowNode } from '@xyflow/react'
import '@xyflow/react/dist/base.css'
import type { DiffResponse, Opening, SceneResponse, Version } from '../../api'
import { action, componentLevel, componentOf, dataLevel, dataOf, insideOf, source, useApi } from '../../api'
import type { Route } from '../../route'
import { Failure, Loading } from '../../ui/marks'
import { kindStyle } from '../../ui/kinds'
import { Inspector } from '../Inspector'
import { Legend } from './Legend'
import { Ghosts, Markers, edgeTypes, nodeTypes } from './parts'
import type { Delta, Draft, ElementData } from './scene'
import { arrangeDraft, membersOf, toDelta, toFlow } from './scene'
import { SaveView } from './SaveView'

/** "context", "container", or "component:<container id>" — one container's components. */
export type Level = string

const levels: [string, string][] = [
  ['context', 'Context'],
  ['container', 'Containers'],
  ['component', 'Components'],
  ['data', 'Data'],
]

// With no level named, the explorer opens where C4 starts — the system in its context — and each
// level opens onto the next: the system onto its containers, a container onto its components.
const levelOf = (raw?: string): Level => (raw === 'container' || insideOf(raw) ? raw! : 'context')

type Go = (r: Partial<Route>, o?: { keep?: boolean; replace?: boolean }) => void

/**
 * The architecture explorer (surface-spec §5.2). The app draws the scene the engine stored — same
 * elements, positions and routes as the committed SVG — with selection, focus, search and zoom.
 * In the live app a person can drag boxes and boundaries into a readable arrangement and keep it
 * in layout.yaml (S-6); the preview while dragging is the engine's own algorithm, and once saved
 * the engine draws it everywhere. Level, selection, search and focus live in the URL, which is
 * what a saved view (S-7) records.
 */
export function Explorer(props: {
  version: number | null
  versions: Version[]
  route: Route
  go: Go
  editable: boolean
  onViewsChanged: () => void
}) {
  const level = levelOf(props.route.level)
  const [reload, setReload] = useState(0)
  const q = props.version ? `&version=${props.version}` : ''
  const scene = useApi<SceneResponse>(`/api/scene?view=${encodeURIComponent(level)}${q}${reload ? `#${reload}` : ''}`)
  const selected = props.route.focus ?? null
  // Only the latest version can be arranged: an arrangement is for the architecture as it is now.
  const editable = props.editable && props.version === null

  // The change overlay: the diff against the compared version, and that version's scene for ghosts.
  const current = scene.data?.version
  const compare = props.route.from !== undefined && props.route.from !== current ? props.route.from : undefined
  const diff = useApi<DiffResponse>(compare !== undefined && current !== undefined ? `/api/diff?from=${compare}&to=${current}` : null)
  const before = useApi<SceneResponse>(compare !== undefined ? `/api/scene?view=${encodeURIComponent(level)}&version=${compare}` : null)
  const delta = useMemo(() => (diff.data ? toDelta(diff.data, before.data) : undefined), [diff.data, before.data])

  const select = (id: string | null) =>
    props.go({ screen: 'explorer', level: props.route.level, focus: id ?? undefined, q: props.route.q, dim: props.route.dim, from: props.route.from }, { replace: true })

  return (
    <div className="explorer">
      <Toolbar
        level={level}
        version={props.version}
        versions={props.versions}
        current={current}
        go={props.go}
        route={props.route}
        editable={editable}
        scene={scene.data}
        onViewsChanged={props.onViewsChanged}
      />
      {scene.error && (
        <div className="explorer-state">
          <Failure error={scene.error} />
        </div>
      )}
      {!scene.data && !scene.error && (
        <div className="explorer-state">
          <Loading />
        </div>
      )}
      {scene.data && (
        <ReactFlowProvider>
          <Arranged
            key={`${level}-${props.version ?? 'latest'}`}
            scene={scene.data}
            route={props.route}
            go={props.go}
            selected={selected}
            onSelect={select}
            editable={editable}
            delta={delta}
            reload={() => setReload((n) => n + 1)}
            open={(id) =>
              level === 'context'
                ? props.go({ screen: 'explorer', level: 'container', from: props.route.from })
                : props.go({ screen: 'explorer', level: componentLevel(id), from: props.route.from })
            }
          />
        </ReactFlowProvider>
      )}
    </div>
  )
}

function Toolbar(props: {
  level: Level
  version: number | null
  versions: Version[]
  current?: number
  go: Go
  route: Route
  editable: boolean
  scene?: SceneResponse
  onViewsChanged: () => void
}) {
  const q = props.version ? `&version=${props.version}` : ''
  const [saving, setSaving] = useState(false)
  // The Components and Data tabs open the selected container when it has them, else the one on
  // screen, else the first that does.
  const lens = componentOf(props.level) ? 'component' : dataOf(props.level) ? 'data' : props.level
  const inside = insideOf(props.level)
  const openings = { component: props.scene?.components ?? [], data: props.scene?.data ?? [] }
  const targetOf = (list: Opening[]) => list.find((o) => o.id === props.route.focus)?.id ?? list.find((o) => o.id === inside)?.id ?? list[0]?.id
  const target = { component: targetOf(openings.component), data: targetOf(openings.data) }
  const tab = lens
  return (
    <div className="explorer-toolbar">
      <div className="segmented" role="tablist" aria-label="C4 level">
        {levels.map(([id, label]) => (
          <button
            key={id}
            role="tab"
            aria-selected={tab === id}
            className={tab === id ? 'on' : ''}
            disabled={(id === 'component' || id === 'data') && !target[id]}
            title={id === 'component' && !target.component ? 'No container’s code was read' : id === 'data' && !target.data ? 'No code declares a table' : undefined}
            onClick={() =>
              id === 'component' || id === 'data'
                ? target[id] && props.go({ screen: 'explorer', level: (id === 'data' ? dataLevel : componentLevel)(target[id]!), from: props.route.from })
                : props.go({ screen: 'explorer', level: id }, { keep: true })
            }
          >
            {label}
          </button>
        ))}
      </div>
      {inside && (lens === 'component' || lens === 'data') && (
        <ContainerPicker
          openings={openings[lens]}
          current={inside}
          go={props.go}
          route={props.route}
          level={lens === 'data' ? dataLevel : componentLevel}
          noun={lens === 'data' ? 'tables' : 'components'}
        />
      )}
      <label className={`tool compare ${props.route.from !== undefined ? 'on' : ''}`} title="Mark on the diagram what changed since another version">
        <span className="mono strong">Δ</span>
        <select
          value={props.route.from ?? ''}
          onChange={(e) => props.go({ screen: 'explorer', from: e.target.value ? Number(e.target.value) : undefined }, { keep: true, replace: true })}
        >
          <option value="">Compare with…</option>
          {[...props.versions]
            .sort((a, b) => b.id - a.id)
            .filter((v) => v.id !== props.current)
            .map((v) => (
              <option key={v.id} value={v.id}>
                v{v.id} · {new Date(v.created_at).toLocaleDateString(undefined, { day: 'numeric', month: 'short' })}
                {v.commit ? ` · ${v.commit.slice(0, 7)}` : ''}
              </option>
            ))}
        </select>
      </label>
      <div className="toolbar-gap" />
      {props.editable && (
        <button className="tool" onClick={() => setSaving(true)} title="Name this view — level, selection, search and focus — and keep it in views.yaml">
          Save view…
        </button>
      )}
      <a
        className="tool"
        href={source(`/api/svg?view=${encodeURIComponent(props.level)}${q}`)}
        download={`${props.level.replace(/^(component|data):[^:]*:/, '$1-').replace(/[:/]/g, '-')}.svg`}
        title="The same scene, as the committed SVG"
      >
        Export SVG
      </a>
      {saving && (
        <SaveView
          route={props.route}
          level={props.level}
          placed={props.scene?.arrangement.placed.length ?? 0}
          onClose={() => setSaving(false)}
          onSaved={() => {
            setSaving(false)
            props.onViewsChanged()
          }}
        />
      )}
    </div>
  )
}

/** At the component level: which container's inside is shown, and the others whose code was read. */
function ContainerPicker(props: { openings: Opening[]; current: string; go: Go; route: Route; level: (id: string) => string; noun: string }) {
  return (
    <label className="tool compare" title={`The container whose ${props.noun} are shown`}>
      <span className="muted">inside</span>
      <select value={props.current} onChange={(e) => props.go({ screen: 'explorer', level: props.level(e.target.value), from: props.route.from })}>
        {props.openings.map((o) => (
          <option key={o.id} value={o.id}>
            {o.name} · {o.components} {props.noun}
          </option>
        ))}
      </select>
    </label>
  )
}

/** The arrangement bar, the canvas and the inspector, sharing one draft. */
function Arranged(props: {
  scene: SceneResponse
  route: Route
  go: Go
  selected: string | null
  onSelect: (id: string | null) => void
  editable: boolean
  delta?: Delta
  reload: () => void
  open: (container: string) => void
}) {
  const { scene } = props
  const opens = useMemo(() => {
    const m = new Map(scene.components.map((o) => [o.id, o.components]))
    // In the context view, the system box opens onto the containers.
    if (scene.view === 'context') for (const n of scene.model.nodes ?? []) if (n.kind === 'system') m.set(n.id, scene.containers ?? 0)
    return m
  }, [scene])
  const holds = useMemo(() => new Map((scene.data ?? []).map((o) => [o.id, o.components])), [scene.data])
  const [draft, setDraft] = useState<Draft>({})
  const [busy, setBusy] = useState(false)
  const [problem, setProblem] = useState<{ text: string; conflict: boolean } | null>(null)
  const moved = Object.keys(draft)

  const layout = useMemo(() => arrangeDraft(scene.layout, draft), [scene.layout, draft])
  // The corner mark shows what was moved and not yet saved; once saved, every box of an arranged
  // view keeps its position, so marking them all would say nothing.
  const placed = useMemo(() => new Set(Object.keys(draft)), [draft])
  const isNew = useMemo(
    () => new Set(scene.arrangement.placed.length ? scene.arrangement.new.filter((id) => !draft[id]) : []),
    [scene.arrangement, draft],
  )

  const run = async (path: string, body: unknown) => {
    setBusy(true)
    setProblem(null)
    try {
      await action(path, body)
      setDraft({})
      props.reload()
    } catch (e) {
      const text = (e as Error).message
      setProblem({ text, conflict: /changed on disk/.test(text) })
    } finally {
      setBusy(false)
    }
  }

  return (
    <>
      <ArrangementBar
        scene={scene}
        moved={moved.length}
        editable={props.editable}
        busy={busy}
        problem={problem}
        onSave={() => run('/api/layout', { view: scene.view, positions: draft, base: scene.arrangement.hash })}
        onDiscard={() => {
          setDraft({})
          setProblem(null)
        }}
        onReset={() => run('/api/layout/reset', { view: scene.view, base: scene.arrangement.hash })}
        onReload={() => {
          setDraft({})
          setProblem(null)
          props.reload()
        }}
      />
      <div className={`explorer-body ${props.selected ? '' : 'nothing-selected'}`}>
        <div className="explorer-canvas">
          <Canvas
            scene={scene}
            layout={layout}
            placed={placed}
            isNew={isNew}
            route={props.route}
            go={props.go}
            selected={props.selected}
            onSelect={props.onSelect}
            editable={props.editable}
            setDraft={setDraft}
            delta={props.delta}
            opens={opens}
            open={props.open}
          />
        </div>
        <Inspector
          model={scene.model}
          id={props.selected}
          onSelect={props.onSelect}
          opens={opens}
          onOpen={props.open}
          holds={holds}
          unresolved={scene.unresolved ?? []}
          onPage={(id) => props.go({ screen: 'component', focus: id })}
          onOpenData={(id) => props.go({ screen: 'explorer', level: dataLevel(id), from: props.route.from })}
          changes={props.selected && props.delta ? { from: props.delta.from, mark: props.delta.nodes.get(props.selected) ?? props.delta.edges.get(props.selected), list: props.delta.changes.get(props.selected) ?? [] } : undefined}
        />
      </div>
      <StatusBar scene={scene} selected={props.selected} route={props.route} delta={props.delta} />
    </>
  )
}

function ArrangementBar(props: {
  scene: SceneResponse
  moved: number
  editable: boolean
  busy: boolean
  problem: { text: string; conflict: boolean } | null
  onSave: () => void
  onDiscard: () => void
  onReset: () => void
  onReload: () => void
}) {
  const a = props.scene.arrangement
  const saved = a.placed.length > 0
  const dirty = props.moved > 0
  const title = dirty ? 'Arrangement changed' : saved ? `Arrangement · ${a.file}` : 'Automatic layout'
  const detail = dirty
    ? `${props.moved} ${props.moved === 1 ? 'box' : 'boxes'} moved · not saved`
    : saved
      ? `${a.placed.length} ${a.placed.length === 1 ? 'position' : 'positions'} kept${a.new.length ? ` · ${a.new.length} new since, placed automatically` : ''}`
      : props.editable
        ? 'positions from the engine · drag a box or a boundary to arrange'
        : 'positions from the engine'

  return (
    <div className={`arrangement-bar ${dirty ? 'dirty' : ''}`}>
      <span className="arr-title">
        <span className="placed-mark" aria-hidden="true" />
        {title}
      </span>
      <span className="arr-detail">{detail}</span>
      {a.stale.length > 0 && !dirty && (
        <span className="arr-stale" title="Like an unmatched rule: these saved positions name elements this view no longer has">
          ! {a.stale.length} saved {a.stale.length === 1 ? 'position matches' : 'positions match'} nothing: {a.stale.join(', ')}
        </span>
      )}
      {props.problem && (
        <span className="arr-problem">
          {props.problem.text}
          {props.problem.conflict && (
            <button className="text-link" onClick={props.onReload}>
              Reload
            </button>
          )}
        </span>
      )}
      <span className="arr-actions">
        {dirty && (
          <>
            <button className="tool" onClick={props.onDiscard} disabled={props.busy}>
              Discard
            </button>
            <button className="tool on" onClick={props.onSave} disabled={props.busy} title="Writes .archdoc/layout.yaml — commit it to share the arrangement">
              Save to layout.yaml
            </button>
          </>
        )}
        {props.editable && saved && (
          <button className="text-link" onClick={props.onReset} disabled={props.busy} title="Forget this view's saved positions">
            Reset to automatic layout
          </button>
        )}
      </span>
    </div>
  )
}

type DragStart = { members: string[]; origin: { x: number; y: number }; base: Record<string, { x: number; y: number }> }

function Canvas(props: {
  scene: SceneResponse
  layout: SceneResponse['layout']
  placed: Set<string>
  isNew: Set<string>
  route: Route
  go: Go
  selected: string | null
  onSelect: (id: string | null) => void
  editable: boolean
  setDraft: (d: Draft | ((d: Draft) => Draft)) => void
  delta?: Delta
  opens: Map<string, number>
  open: (container: string) => void
}) {
  const { scene, route, selected } = props
  const find = route.q ?? ''
  const focus = route.dim === '1'
  const [legend, setLegend] = useState(false)
  const findRef = useRef<HTMLInputElement>(null)
  const flow = useReactFlow()
  const dragStart = useRef<DragStart | null>(null)

  const { nodes, edges } = useMemo(
    () => toFlow(scene, props.layout, { selected, focus, find, placed: props.placed, isNew: props.isNew, editable: props.editable, delta: props.delta, opens: props.opens }),
    [scene, props.layout, selected, focus, find, props.placed, props.isNew, props.editable, props.delta, props.opens],
  )

  const setRoute = (patch: Partial<Route>) =>
    props.go({ screen: 'explorer', level: route.level, focus: route.focus, q: route.q, dim: route.dim, from: route.from, ...patch }, { replace: true })

  // A new scene — another level or version — is framed whole, unless the address names a selection:
  // a shared link to one element opens on that element.
  useEffect(() => {
    const t = setTimeout(() => {
      const target = selected && !selected.includes('>') && (scene.model.nodes ?? []).some((n) => n.id === selected)
      if (target) flow.fitView({ nodes: [{ id: selected! }], padding: 1.2, duration: 0, maxZoom: 1.2 })
      else flow.fitView({ padding: 0.08, duration: 0, maxZoom: 1.2 })
    }, 0)
    return () => clearTimeout(t)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [scene.version, scene.view, flow])

  // "/" jumps to the search.
  useEffect(() => {
    const on = (e: KeyboardEvent) => {
      if (e.key === '/' && !(e.target as HTMLElement).closest('input, select, textarea')) {
        e.preventDefault()
        findRef.current?.focus()
      }
    }
    window.addEventListener('keydown', on)
    return () => window.removeEventListener('keydown', on)
  }, [])

  const jumpToMatch = () => {
    const hit = nodes.find((n) => n.type === 'element' && (n.data as ElementData).match)
    if (!hit) return
    props.onSelect(hit.id)
    flow.fitView({ nodes: [{ id: hit.id }], padding: 1.2, duration: 200, maxZoom: 1.4 })
  }

  // Dragging a boundary moves the boxes it contained, from where they were when the drag began.
  const startDrag = (n: FlowNode) => {
    if (n.type !== 'boundary') return
    const members = membersOf(scene.layout, n.id.slice('boundary:'.length))
    const base: DragStart['base'] = {}
    for (const id of members) {
      const r = (props.layout.boxes ?? []).find((b) => b.id === id)?.rect
      if (r) base[id] = { x: r.x, y: r.y }
    }
    dragStart.current = { members, origin: { ...n.position }, base }
  }
  const drag = (n: FlowNode) => {
    if (n.type === 'element') {
      props.setDraft((d) => ({ ...d, [n.id]: { x: Math.round(n.position.x), y: Math.round(n.position.y) } }))
      return
    }
    const s = dragStart.current
    if (n.type !== 'boundary' || !s) return
    const dx = n.position.x - s.origin.x
    const dy = n.position.y - s.origin.y
    props.setDraft((d) => {
      const next = { ...d }
      for (const id of s.members) next[id] = { x: Math.round(s.base[id].x + dx), y: Math.round(s.base[id].y + dy) }
      return next
    })
  }

  return (
    <>
      <Markers />
      <div className="canvas-tools">
        <label className="find">
          <span>/</span>
          <input
            ref={findRef}
            value={find}
            placeholder="Find node"
            onChange={(e) => setRoute({ q: e.target.value || undefined })}
            onKeyDown={(e) => {
              if (e.key === 'Enter') jumpToMatch()
              if (e.key === 'Escape') {
                setRoute({ q: undefined })
                e.currentTarget.blur()
                e.stopPropagation()
              }
            }}
          />
        </label>
        <button
          className={`tool ${focus ? 'on' : ''}`}
          onClick={() => setRoute({ dim: focus ? undefined : '1' })}
          disabled={!selected}
          title={selected ? 'Dim everything not connected to the selection' : 'Select a box first'}
        >
          Focus
        </button>
      </div>

      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        edgeTypes={edgeTypes}
        onNodeClick={(_, n) => n.type === 'element' && props.onSelect(n.id)}
        onNodeDoubleClick={(_, n) => n.type === 'element' && props.opens.has(n.id) && props.open(n.id)}
        zoomOnDoubleClick={false}
        onEdgeClick={(_, e) => props.onSelect(e.id)}
        onPaneClick={() => props.onSelect(null)}
        onNodeDragStart={(_, n) => startDrag(n)}
        onNodeDrag={(_, n) => drag(n)}
        onNodeDragStop={() => {
          dragStart.current = null
        }}
        nodesDraggable={props.editable}
        nodesConnectable={false}
        elementsSelectable
        minZoom={0.1}
        maxZoom={2.5}
        proOptions={{ hideAttribution: true }}
      >
        <Background variant={BackgroundVariant.Dots} gap={18} size={1.2} color="var(--canvas-dot)" />
        {props.delta && (
          <ViewportPortal>
            <Ghosts delta={props.delta} />
          </ViewportPortal>
        )}
        <MiniMap
          pannable
          zoomable
          nodeColor={(n: FlowNode) => (n.type === 'element' ? `var(--k-${kindStyle((n.data as ElementData).node.kind).hue})` : 'transparent')}
          nodeStrokeColor="transparent"
          maskColor="rgba(227, 230, 229, 0.6)"
          className="minimap"
          style={{ right: 58, bottom: 14, margin: 0, width: 150, height: 84 }}
          ariaLabel="Overview of the whole diagram"
        />
        <ZoomControls editable={props.editable} />
      </ReactFlow>

      <button className="legend-toggle" onClick={() => setLegend((l) => !l)}>
        Legend
      </button>
      {legend && <Legend onClose={() => setLegend(false)} />}
    </>
  )
}

function ZoomControls({ editable }: { editable: boolean }) {
  const flow = useReactFlow()
  const zoom = useStore((s) => s.transform[2])
  return (
    <>
      <span className="zoom-readout">
        {Math.round(zoom * 100)}% · {editable ? 'drag the canvas to pan, a box to arrange' : 'drag to pan'}
      </span>
      <div className="zoom">
        <div className="zoom-buttons">
          <button onClick={() => flow.zoomIn({ duration: 120 })} aria-label="Zoom in">
            +
          </button>
          <button onClick={() => flow.zoomOut({ duration: 120 })} aria-label="Zoom out">
            −
          </button>
          <button onClick={() => flow.fitView({ padding: 0.08, duration: 160, maxZoom: 1.2 })} aria-label="Fit">
            FIT
          </button>
        </div>
      </div>
    </>
  )
}

function StatusBar({ scene, selected, route, delta }: { scene: SceneResponse; selected: string | null; route: Route; delta?: Delta }) {
  const n = scene.model.nodes?.length ?? 0
  const e = scene.model.edges?.length ?? 0
  const parts = componentOf(scene.view)
  const tables = dataOf(scene.view)
  const url = [`level=${scene.view}`, selected && `focus=${selected}`, route.q && `q=${route.q}`, route.dim && 'dim=1', route.from !== undefined && `from=${route.from}`]
    .filter(Boolean)
    .join('&')
  return (
    <div className="explorer-status">
      <span>
        {parts
          ? `${n} components of ${scene.model.name} · ${e} uses, each an import`
          : tables
            ? `${n} tables of ${scene.model.name} · ${e} foreign keys`
            : `${n} elements · ${e} relationships`}
      </span>
      <span>v{scene.version}</span>
      {delta && (
        <span className="status-delta">
          Δ v{delta.from} → v{delta.to} · +{delta.count.added} · ±{delta.count.changed} · −{delta.count.removed} · {delta.count.words} words only
        </span>
      )}
      <span className="status-url">#{url}</span>
    </div>
  )
}
