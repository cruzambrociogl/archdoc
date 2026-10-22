import { useEffect, useMemo, useRef, useState } from 'react'
import { Background, BackgroundVariant, MiniMap, ReactFlow, ReactFlowProvider, ViewportPortal, useReactFlow, useStore } from '@xyflow/react'
import type { Node as FlowNode } from '@xyflow/react'
import '@xyflow/react/dist/base.css'
import type { DiffResponse, ModelResponse, Opening, SavedView, SceneResponse, Version } from '../../api'
import { componentLevel, componentOf, dataLevel, dataOf, insideOf, mainLevel, mainOf, overview, structureLevel, structureOf, useApi } from '../../api'
import { useSource } from '../../source'
import type { Route } from '../../route'
import { Planned } from '../../ui/Planned'
import { Failure, Loading } from '../../ui/marks'
import { kindStyle } from '../../ui/kinds'
import { Inspector } from '../Inspector'
import { Gone } from '../Gone'
import { Legend } from './Legend'
import { Ghosts, Markers, edgeTypes, nodeTypes } from './parts'
import type { Delta, Draft, ElementData } from './scene'
import { arrangeDraft, membersOf, toDelta, toFlow } from './scene'
import { SaveView } from './SaveView'

/** "context", "container", or "component:<container id>" — one container's components. */
export type Level = string

// Zooming and fitting animate, unless the reader asked for less motion (§4.8).
const moving = (ms: number) => (window.matchMedia?.('(prefers-reduced-motion: reduce)').matches ? 0 : ms)

// With no level named, the explorer opens where C4 starts — the system in its context — and each
// level opens onto the next: the system onto its containers, a container onto its components.
const levelOf = (raw?: string): Level => (raw === 'container' || raw === 'dataflow' || insideOf(raw) ? raw! : 'context')

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
  views: SavedView[]
  onViewsChanged: () => void
  /** Tells the shell each part's name as the Map shows it, for the breadcrumb. */
  onNames?: (names: Record<string, string>) => void
}) {
  const asked = levelOf(props.route.level)
  const [reload, setReload] = useState(0)
  const q = props.version ? `&version=${props.version}` : ''
  const selected = props.route.focus ?? null
  // A container with more components than a diagram can show opens on its main ones. Every one of
  // them is drawn when that is asked for, or when what is selected is not among the main ones.
  const container = componentOf(asked)
  const main = useApi<SceneResponse>(container && !props.route.all ? `/api/scene?view=${encodeURIComponent(mainLevel(container))}${q}${reload ? `#${reload}` : ''}` : null)
  const tryMain = !!container && !props.route.all && !main.error
  const inMain = !selected || (main.data?.model.nodes ?? []).some((n) => n.id === selected)
  const useMain = tryMain && !!main.data && inMain
  const level = useMain ? mainLevel(container!) : asked
  const sceneName = overview(asked) && props.route.all ? `${asked}:all` : asked
  const full = useApi<SceneResponse>(tryMain && (!main.data || inMain) ? null : `/api/scene?view=${encodeURIComponent(sceneName)}${q}${reload ? `#${reload}` : ''}`)
  // A selection the overview leaves out opens every external system, as one outside the main components does.
  const outside = overview(asked) && !props.route.all && !!selected && !!full.data?.model.folded && !(full.data.model.nodes ?? []).some((n) => n.id === selected)
  useEffect(() => {
    if (outside) props.go({ ...props.route, all: '1' }, { replace: true })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [outside])
  const scene = useMain ? main : tryMain && !main.data ? { data: undefined, error: undefined } : full
  // Only the latest version can be arranged: an arrangement is for the architecture as it is now.
  const editable = props.editable && props.version === null

  // The change overlay: the diff against the compared version, and that version's scene for ghosts.
  const current = scene.data?.version
  const onNames = props.onNames
  useEffect(() => {
    const d = scene.data
    if (!d || !onNames) return
    const names: Record<string, string> = {}
    for (const o of [...(d.components ?? []), ...(d.data ?? []), ...(d.main ?? []), ...(d.structure ?? [])]) names[o.id] = o.name
    onNames(names)
  }, [scene.data, onNames])
  const compare = props.route.from !== undefined && props.route.from !== current ? props.route.from : undefined
  const diff = useApi<DiffResponse>(compare !== undefined && current !== undefined ? `/api/diff?from=${compare}&to=${current}` : null)
  const before = useApi<SceneResponse>(compare !== undefined ? `/api/scene?view=${encodeURIComponent(useMain ? level : sceneName)}&version=${compare}` : null)
  const delta = useMemo(() => (diff.data ? toDelta(diff.data, before.data) : undefined), [diff.data, before.data])

  // A link whose part is in no level of this version: the picture says where it went (§4.8). One in
  // another level is not gone — only a part the whole model lacks is.
  const offScene = !!selected && !selected.includes('>') && !!scene.data && !(scene.data.model.nodes ?? []).some((n) => n.id === selected)
  const whole = useApi<ModelResponse>(offScene ? `/api/model${props.version ? `?version=${props.version}` : ''}` : null)
  const gone = offScene && !!whole.data && !(whole.data.model.nodes ?? []).some((n) => n.id === selected)

  const select = (id: string | null) =>
    props.go({ screen: 'explorer', level: props.route.level, focus: id ?? undefined, q: props.route.q, dim: props.route.dim, all: props.route.all, from: props.route.from }, { replace: true })

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
        views={props.views}
        onViewsChanged={props.onViewsChanged}
      />
      {scene.error && (
        <div className="explorer-state">
          <Failure error={scene.error} />
        </div>
      )}
      {!scene.data && !scene.error && (
        <div className="explorer-state">
          <Loading what="the map" />
        </div>
      )}
      {gone && <Gone kind="part" id={selected!} shown={whole.data!.version} name={whole.data!.model.name} go={props.go} compact />}
      {scene.data && (
        <ReactFlowProvider>
          <Arranged
            key={`${useMain ? level : sceneName}-${props.version ?? 'latest'}`}
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
                ? props.go({ screen: 'explorer', level: scene.data?.tiny && scene.data.components[0] ? componentLevel(scene.data.components[0].id) : 'container', from: props.route.from })
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
  views: SavedView[]
  onViewsChanged: () => void
}) {
  const source = useSource()
  const q = props.version ? `&version=${props.version}` : ''
  const [saving, setSaving] = useState(false)
  const [open, setOpen] = useState<'part' | 'views' | null>(null)
  // A menu opens toward the side with room: when the toolbar wraps, Saved views starts a row.
  const [opensRight, setOpensRight] = useState(false)
  const bar = useRef<HTMLDivElement>(null)
  useEffect(() => {
    if (!open) return
    const out = (e: MouseEvent) => bar.current && !bar.current.contains(e.target as Node) && setOpen(null)
    const esc = (e: KeyboardEvent) => e.key === 'Escape' && setOpen(null)
    document.addEventListener('mousedown', out)
    document.addEventListener('keydown', esc)
    return () => {
      document.removeEventListener('mousedown', out)
      document.removeEventListener('keydown', esc)
    }
  }, [open])

  // Inside one part, and What it stores, open the selected part when it has them, else the one on
  // screen, else the first that does.
  const byFolder = structureOf(props.level) !== undefined
  const onMain = mainOf(props.level) !== undefined
  const lens = componentOf(props.level) || byFolder || onMain ? 'component' : dataOf(props.level) ? 'data' : props.level
  const inside = insideOf(props.level)
  const openings = { component: props.scene?.components ?? [], data: props.scene?.data ?? [] }
  const targetOf = (list: Opening[]) => list.find((o) => o.id === props.route.focus)?.id ?? list.find((o) => o.id === inside)?.id ?? list[0]?.id
  const target = { component: targetOf(openings.component), data: targetOf(openings.data) }
  const hasMain = !!inside && (props.scene?.main ?? []).some((o) => o.id === inside)
  const hasFolders = !!inside && (props.scene?.structure ?? []).some((o) => o.id === inside)
  const allCount = (props.scene?.components ?? []).find((o) => o.id === inside)?.components ?? ''
  const externals = (props.scene?.model.nodes ?? []).filter((n) => n.kind === 'external').length + (props.scene?.model.folded ?? 0)
  const insideName = openings.component.find((o) => o.id === (lens === 'component' ? inside : target.component))?.name

  // Two kinds of move (App Findings, round 4, problem 1): zoom, a path from the outside world inward,
  // whose inside step always names a part; and lenses, other questions about the same system.
  const zoom: { id: string; plain: string; term: string; on?: () => void; picks?: boolean }[] = [
    { id: 'context', plain: 'The outside world', term: 'context', on: () => props.go({ screen: 'explorer', level: 'context' }, { keep: true }) },
    ...(props.scene?.tiny ? [] : [{ id: 'container', plain: 'Apps and services', term: 'containers', on: () => props.go({ screen: 'explorer', level: 'container' }, { keep: true }) }]),
    ...(target.component
      ? [{ id: 'component', plain: `Inside ${insideName ?? 'one part'}`, term: 'components', on: () => props.go({ screen: 'explorer', level: componentLevel(target.component!), from: props.route.from }), picks: openings.component.length > 1 }]
      : []),
  ]
  const lenses: { id: string; plain: string; term: string; on: () => void }[] = [
    ...(target.data ? [{ id: 'data', plain: 'What it stores', term: 'entities', on: () => props.go({ screen: 'explorer', level: dataLevel(target.data!), from: props.route.from }) }] : []),
    ...(props.scene?.dataflow ? [{ id: 'dataflow', plain: 'How data moves', term: 'data flow', on: () => props.go({ screen: 'explorer', level: 'dataflow' }, { keep: true }) }] : []),
  ]
  const shownView = props.views.find((v) => v.level === props.level && (v.focus ?? '') === (props.route.focus ?? '') && (v.find ?? '') === (props.route.q ?? '') && !!v.dim === (props.route.dim === '1'))

  return (
    <div className="explorer-toolbar" ref={bar}>
      <div className="map-group" role="group" aria-label="Zoom">
        <span className="map-group-label">Zoom</span>
        {zoom.map((z, i) => (
          <span key={z.id} className="zoom-step">
            {i > 0 && <span className="crumb-sep">›</span>}
            <button className={`zoom-btn ${lens === z.id ? 'on' : ''}`} aria-pressed={lens === z.id} onClick={() => (z.picks && lens === 'component' ? setOpen(open === 'part' ? null : 'part') : z.on?.())}>
              <span className="zoom-plain">
                {z.plain}
                {z.picks && <span className="caret"> ▾</span>}
              </span>
              <span className="zoom-term">{z.term}</span>
            </button>
          </span>
        ))}
        {open === 'part' && (
          <div className="pop map-pop">
            <div className="pop-q">Go inside</div>
            {openings.component.map((o) => (
              <button
                key={o.id}
                className={`more-item ${o.id === inside ? 'on' : ''}`}
                onClick={() => (setOpen(null), props.go({ screen: 'explorer', level: componentLevel(o.id), from: props.route.from }))}
              >
                <span>
                  <span className="more-label mono">{o.name}</span>
                  <span className="more-sub">{o.components} components</span>
                </span>
              </button>
            ))}
          </div>
        )}
      </div>
      {lenses.length > 0 && (
        <div className="map-group" role="group" aria-label="Lenses">
          <span className="map-divider" />
          <span className="map-group-label">Lens</span>
          {lenses.map((l) => (
            <button key={l.id} className={`lens-btn ${lens === l.id ? 'on' : ''}`} aria-pressed={lens === l.id} onClick={l.on}>
              {l.plain}
              <span className="zoom-term">{l.term}</span>
            </button>
          ))}
          <Planned id="deployment" go={props.go}>
            Where it runs
          </Planned>
        </div>
      )}
      {inside && lens === 'data' && openings.data.length > 1 && (
        <ContainerPicker openings={openings.data} current={inside} go={props.go} route={props.route} level={dataLevel} noun="tables" />
      )}
      {inside && lens === 'component' && (hasMain || hasFolders) && (
        <div className="segmented" role="tablist" aria-label="Which components are shown" title="The same code: the few components that handle the most, every component, or its folders">
          {hasMain && (
            <button className={onMain ? 'on' : ''} role="tab" aria-selected={onMain} onClick={() => props.go({ screen: 'explorer', level: componentLevel(inside), from: props.route.from })}>
              Main
            </button>
          )}
          <button
            className={!onMain && !byFolder ? 'on' : ''}
            role="tab"
            aria-selected={!onMain && !byFolder}
            onClick={() => props.go({ screen: 'explorer', level: componentLevel(inside), all: hasMain ? '1' : undefined, from: props.route.from })}
          >
            {hasMain ? `All ${allCount}` : 'By feature'}
          </button>
          {hasFolders && (
            <button className={byFolder ? 'on' : ''} role="tab" aria-selected={byFolder} onClick={() => props.go({ screen: 'explorer', level: structureLevel(inside), from: props.route.from })}>
              By folder
            </button>
          )}
        </div>
      )}
      {overview(props.level) && (props.route.all || !!props.scene?.model.folded) && (
        <div className="segmented" role="tablist" aria-label="Which external systems are shown" title="The external systems the most containers reach, or every one">
          <button className={!props.route.all ? 'on' : ''} role="tab" aria-selected={!props.route.all} onClick={() => props.go({ screen: 'explorer', level: props.level, from: props.route.from })}>
            Main
          </button>
          <button className={props.route.all ? 'on' : ''} role="tab" aria-selected={!!props.route.all} onClick={() => props.go({ screen: 'explorer', level: props.level, all: '1', from: props.route.from })}>
            All{externals ? ` ${externals}` : ''}
          </button>
        </div>
      )}
      <div className="toolbar-gap" />
      {/* Compare and Saved views sit apart from both groups (App Findings, round 4). */}
      {props.versions.length > 1 && (
        <label className={`tool compare ${props.route.from !== undefined ? 'on' : ''}`} title="Mark on the map what changed since another version">
          <span className="strong">Compare</span>
          <select
            value={props.route.from ?? ''}
            aria-label="Compare with"
            onChange={(e) => props.go({ screen: 'explorer', from: e.target.value ? Number(e.target.value) : undefined }, { keep: true, replace: true })}
          >
            <option value="">with…</option>
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
      )}
      {(props.views.length > 0 || props.editable) && (
        <div className="pop-anchor">
          <button
            className={`tool ${open === 'views' ? 'on' : ''}`}
            onClick={(e) => {
              const r = e.currentTarget.getBoundingClientRect()
              const room = bar.current?.getBoundingClientRect()
              setOpensRight(!!room && r.right - room.left < 300)
              setOpen(open === 'views' ? null : 'views')
            }}
            aria-expanded={open === 'views'}
          >
            {shownView ? shownView.name : 'Saved views'} ▾
          </button>
          {open === 'views' && (
            <div className={`pop map-pop views-pop ${opensRight ? 'opens-right' : ''}`}>
              <div className="pop-q">Saved views</div>
              {props.views.length === 0 && <p className="more-sub views-none">None yet: arrange the map, then save it with a name.</p>}
              {props.views.map((v) => (
                <button
                  key={v.name}
                  className={`more-item ${v === shownView ? 'on' : ''}`}
                  onClick={() => (setOpen(null), props.go({ screen: 'explorer', level: v.level, focus: v.focus, q: v.find, dim: v.dim ? '1' : undefined }))}
                >
                  <span className="more-label">{v.name}</span>
                </button>
              ))}
              {props.editable && (
                <button className="more-item views-save" onClick={() => (setOpen(null), setSaving(true))}>
                  <span className="more-label">Save this view…</span>
                </button>
              )}
            </div>
          )}
        </div>
      )}
      <Planned id="share-card" go={props.go}>
        Share card
      </Planned>
      <a
        className="tool"
        href={source.url(`/api/svg?view=${encodeURIComponent(props.level)}${q}`)}
        download={`${props.level.replace(/^(component|data|structure|main):[^:]*:/, '$1-').replace(/[:/]/g, '-')}.svg`}
        title="The same picture, as the committed SVG"
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
  const source = useSource()
  const { scene } = props
  const opens = useMemo(() => {
    const m = new Map(scene.components.map((o) => [o.id, o.components]))
    // In the context view, the system box opens onto the containers.
    if (scene.view === 'context' || scene.view === 'context:all') for (const n of scene.model.nodes ?? []) if (n.kind === 'system') m.set(n.id, scene.containers ?? 0)
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
      await source.act(path, body)
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
          go={props.go}
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
    flow.fitView({ nodes: [{ id: hit.id }], padding: 1.2, duration: moving(200), maxZoom: 1.4 })
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
          <button onClick={() => flow.zoomIn({ duration: moving(120) })} aria-label="Zoom in">
            +
          </button>
          <button onClick={() => flow.zoomOut({ duration: moving(120) })} aria-label="Zoom out">
            −
          </button>
          <button onClick={() => flow.fitView({ padding: 0.08, duration: moving(160), maxZoom: 1.2 })} aria-label="Fit">
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
  const parts = componentOf(scene.view) ?? structureOf(scene.view) ?? mainOf(scene.view)
  const ofAll = scene.components.find((o) => o.id === mainOf(scene.view))?.components
  const tables = dataOf(scene.view)
  const url = [`level=${scene.view}`, selected && `focus=${selected}`, route.q && `q=${route.q}`, route.dim && 'dim=1', route.from !== undefined && `from=${route.from}`]
    .filter(Boolean)
    .join('&')
  return (
    <div className="explorer-status">
      <span>
        {parts
          ? mainOf(scene.view)
            ? `${n} of ${scene.model.name}'s ${ofAll} components — its main ones, as the boundary says · ${e} uses drawn, each one's strongest`
            : `${n} ${structureOf(scene.view) ? 'folders' : 'components'} of ${scene.model.name} · ${e} uses, each an import`
          : tables
            ? `${n} tables of ${scene.model.name} · ${e} foreign keys`
            : `${n} elements · ${e} relationships${scene.model.folded ? ` · ${scene.model.folded} external systems not drawn, the ones the fewest containers reach` : ''}`}
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
