import { useEffect, useMemo, useRef, useState } from 'react'
import { Background, BackgroundVariant, MiniMap, ReactFlow, ReactFlowProvider, useReactFlow, useStore } from '@xyflow/react'
import type { Node as FlowNode } from '@xyflow/react'
import '@xyflow/react/dist/base.css'
import type { SceneResponse } from '../../api'
import { useApi } from '../../api'
import { Failure, Loading } from '../../ui/marks'
import { kindStyle } from '../../ui/kinds'
import { Inspector } from '../Inspector'
import { Legend } from './Legend'
import { Markers, edgeTypes, nodeTypes } from './parts'
import type { ElementData } from './scene'
import { toFlow } from './scene'

export type Level = 'context' | 'container'

const levels: [Level, string][] = [
  ['context', 'Context'],
  ['container', 'Containers'],
]

/**
 * The architecture explorer (surface-spec §5.2). The app draws the diagram itself, from the
 * scene the engine stored: same elements, same positions, same routes as the committed SVG, now
 * with selection, focus, search and zoom. Arranging and the change overlay arrive next.
 */
export function Explorer(props: {
  version: number | null
  level: Level
  selected: string | null
  onLevel: (l: Level) => void
  onSelect: (id: string | null) => void
}) {
  const q = props.version ? `&version=${props.version}` : ''
  const scene = useApi<SceneResponse>(`/api/scene?view=${props.level}${q}`)

  return (
    <div className="explorer">
      <Toolbar {...props} />
      <div className={`explorer-body ${props.selected ? '' : 'nothing-selected'}`}>
        <div className="explorer-canvas">
          {scene.error && <div className="explorer-state"><Failure error={scene.error} /></div>}
          {!scene.data && !scene.error && <div className="explorer-state"><Loading /></div>}
          {scene.data && (
            <ReactFlowProvider>
              <Canvas scene={scene.data} selected={props.selected} onSelect={props.onSelect} />
            </ReactFlowProvider>
          )}
        </div>
        {scene.data && <Inspector model={scene.data.model} id={props.selected} onSelect={props.onSelect} />}
      </div>
      {scene.data && <StatusBar scene={scene.data} selected={props.selected} />}
    </div>
  )
}

function Toolbar(props: { level: Level; onLevel: (l: Level) => void; version: number | null }) {
  const q = props.version ? `&version=${props.version}` : ''
  return (
    <div className="explorer-toolbar">
      <div className="segmented" role="tablist" aria-label="C4 level">
        {levels.map(([id, label]) => (
          <button key={id} role="tab" aria-selected={props.level === id} className={props.level === id ? 'on' : ''} onClick={() => props.onLevel(id)}>
            {label}
          </button>
        ))}
      </div>
      <div className="toolbar-gap" />
      <a className="tool" href={`/api/svg?view=${props.level}${q}`} download={`${props.level}.svg`} title="The same scene, as the committed SVG">
        Export SVG
      </a>
    </div>
  )
}

function Canvas({ scene, selected, onSelect }: { scene: SceneResponse; selected: string | null; onSelect: (id: string | null) => void }) {
  const [focus, setFocus] = useState(false)
  const [find, setFind] = useState('')
  const [legend, setLegend] = useState(false)
  const findRef = useRef<HTMLInputElement>(null)
  const flow = useReactFlow()

  const { nodes, edges } = useMemo(() => toFlow(scene, { selected, focus, find }), [scene, selected, focus, find])

  // A new scene — another level or version — is framed whole.
  useEffect(() => {
    const t = setTimeout(() => flow.fitView({ padding: 0.08, duration: 0, maxZoom: 1.2 }), 0)
    return () => clearTimeout(t)
  }, [scene, flow])

  // "/" jumps to the search, as everywhere in the app.
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
    onSelect(hit.id)
    flow.fitView({ nodes: [{ id: hit.id }], padding: 1.2, duration: 200, maxZoom: 1.4 })
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
            onChange={(e) => setFind(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') jumpToMatch()
              if (e.key === 'Escape') {
                setFind('')
                e.currentTarget.blur()
                e.stopPropagation()
              }
            }}
          />
        </label>
        <button
          className={`tool ${focus ? 'on' : ''}`}
          onClick={() => setFocus((f) => !f)}
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
        onNodeClick={(_, n) => n.type === 'element' && onSelect(n.id)}
        onEdgeClick={(_, e) => onSelect(e.id)}
        onPaneClick={() => onSelect(null)}
        nodesDraggable={false}
        nodesConnectable={false}
        elementsSelectable
        minZoom={0.1}
        maxZoom={2.5}
        fitView
        fitViewOptions={{ padding: 0.08, maxZoom: 1.2 }}
        proOptions={{ hideAttribution: true }}
      >
        <Background variant={BackgroundVariant.Dots} gap={18} size={1.2} color="var(--canvas-dot)" />
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
        <ZoomControls />
      </ReactFlow>

      <button className="legend-toggle" onClick={() => setLegend((l) => !l)}>
        Legend
      </button>
      {legend && <Legend onClose={() => setLegend(false)} />}
    </>
  )
}

function ZoomControls() {
  const flow = useReactFlow()
  const zoom = useStore((s) => s.transform[2])
  return (
    <>
    <span className="zoom-readout">{Math.round(zoom * 100)}% · drag to pan</span>
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

function StatusBar({ scene, selected }: { scene: SceneResponse; selected: string | null }) {
  const n = scene.model.nodes?.length ?? 0
  const e = scene.model.edges?.length ?? 0
  return (
    <div className="explorer-status">
      <span>
        {n} elements · {e} relationships
      </span>
      <span>v{scene.version}</span>
      <span className="status-url">
        #level={scene.view}
        {selected ? `&focus=${selected}` : ''}
      </span>
    </div>
  )
}
