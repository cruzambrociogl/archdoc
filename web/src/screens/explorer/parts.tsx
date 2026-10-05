// The diagram's parts, drawn as components from the design tokens (surface-spec §6.4): an
// element box, a boundary, and a relationship routed along the engine's stored curve.

import { BaseEdge, EdgeLabelRenderer, Handle, Position, useStore } from '@xyflow/react'
import type { EdgeProps, NodeProps, Node as FlowNode, Edge as FlowEdge } from '@xyflow/react'
import { kindStyle } from '../../ui/kinds'
import type { BoundaryData, ElementData, RouteData } from './scene'

// Below this zoom a box shows only its name and icon (semantic zoom, "Surface Foundations" 6.4).
const DETAIL_ZOOM = 0.85
const zoomSelector = (s: { transform: [number, number, number] }) => s.transform[2] >= DETAIL_ZOOM

export function ElementNode({ data, selected }: NodeProps<FlowNode<ElementData>>) {
  const n = data.node
  const k = kindStyle(n.kind)
  const detail = useStore(zoomSelector)
  const interp = n.description_provenance?.origin === 'model'
  const tech = n.technology || k.label

  return (
    <div
      className={`el el-${k.hue} ${n.evidence === 'referenced' ? 'el-referenced' : ''} ${selected ? 'el-selected' : ''} ${data.dim ? 'is-dim' : ''} ${data.match ? 'el-match' : ''}`}
      title={`${n.name} · ${k.label}${n.technology ? ` · ${n.technology}` : ''}`}
    >
      {/* Edges follow stored routes; these handles only satisfy the graph library. */}
      <Handle type="target" position={Position.Top} className="el-handle" isConnectable={false} />
      <Handle type="source" position={Position.Bottom} className="el-handle" isConnectable={false} />
      <div className="el-cap" />
      <div className="el-body">
        <div className="el-head">
          <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinejoin="round" strokeLinecap="round">
            <path d={k.icon} />
          </svg>
          <span className="el-name">{n.name}</span>
        </div>
        {detail && <div className="el-tech">{tech}</div>}
        {detail && n.description && (
          <div className={`el-desc ${interp ? 'el-desc-interp' : ''}`}>
            {interp && <span className="mark mark-interpreted" />}
            <span>{n.description}</span>
          </div>
        )}
      </div>
    </div>
  )
}

export function BoundaryNode({ data }: NodeProps<FlowNode<BoundaryData>>) {
  return (
    <div className={`boundary ${data.system ? 'boundary-system' : 'boundary-network'}`}>
      <span className="boundary-label">{data.label}</span>
    </div>
  )
}

export function RoutedEdge({ id, data, selected }: EdgeProps<FlowEdge<RouteData>>) {
  if (!data) return null
  const e = data.edge
  const interp = e.label_provenance?.origin === 'model'
  const text = [e.label, e.technology].filter(Boolean).join(' · ')
  return (
    <>
      <BaseEdge
        id={id}
        path={data.d}
        className={`route ${selected ? 'route-selected' : ''} ${data.dim ? 'is-dim' : ''}`}
        markerEnd={selected ? 'url(#ad-arrow-selected)' : 'url(#ad-arrow)'}
        interactionWidth={14}
      />
      {text && data.labelAt && (
        <EdgeLabelRenderer>
          <div
            className={`route-label nodrag nopan ${selected ? 'route-label-selected' : ''} ${data.dim ? 'is-dim' : ''}`}
            style={{ transform: `translate(-50%, -50%) translate(${data.labelAt.x}px, ${data.labelAt.y}px)` }}
          >
            {interp && <span className="mark mark-interpreted" />}
            <span className={interp ? 'route-label-interp' : ''}>{text}</span>
          </div>
        </EdgeLabelRenderer>
      )}
    </>
  )
}

/** Arrowheads, defined once for every routed edge. */
export function Markers() {
  return (
    <svg className="ad-markers" aria-hidden="true">
      <defs>
        <marker id="ad-arrow" viewBox="0 0 8 8" refX="7" refY="4" markerWidth="8" markerHeight="8" markerUnits="userSpaceOnUse" orient="auto-start-reverse">
          <path d="M0 0L8 4L0 8z" fill="var(--edge)" />
        </marker>
        <marker id="ad-arrow-selected" viewBox="0 0 8 8" refX="7" refY="4" markerWidth="8" markerHeight="8" markerUnits="userSpaceOnUse" orient="auto-start-reverse">
          <path d="M0 0L8 4L0 8z" fill="var(--canvas-accent)" />
        </marker>
      </defs>
    </svg>
  )
}

export const nodeTypes = { element: ElementNode, boundary: BoundaryNode }
export const edgeTypes = { routed: RoutedEdge }
