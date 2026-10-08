// The diagram's parts, drawn as components from the design tokens (surface-spec §6.4): an
// element box, a boundary, and a relationship routed along the engine's stored curve.

import { BaseEdge, EdgeLabelRenderer, Handle, Position, useStore } from '@xyflow/react'
import type { EdgeProps, NodeProps, Node as FlowNode, Edge as FlowEdge } from '@xyflow/react'
import { kindStyle } from '../../ui/kinds'
import type { BoundaryData, Delta, ElementData, Mark, RouteData } from './scene'

// Below this zoom a box shows only its name and icon (semantic zoom, "Surface Foundations" 6.4).
const DETAIL_ZOOM = 0.85
const zoomSelector = (s: { transform: [number, number, number] }) => s.transform[2] >= DETAIL_ZOOM

/** A table: its name in a header band, then a line per column, as the committed SVG draws it. */
function TableNode({ data, selected }: { data: ElementData; selected: boolean }) {
  const n = data.node
  const cols = n.columns ?? []
  const rows = cols.length > TABLE_ROWS + 1 ? cols.slice(0, TABLE_ROWS) : cols
  return (
    <div
      className={`el el-teal el-table ${selected ? 'el-selected' : ''} ${data.dim ? 'is-dim' : ''} ${data.match ? 'el-match' : ''} ${data.delta ? `el-delta-${data.delta}` : ''}`}
      title={`${n.name} · ${cols.length} columns`}
    >
      <Handle type="target" position={Position.Top} className="el-handle" isConnectable={false} />
      <Handle type="source" position={Position.Bottom} className="el-handle" isConnectable={false} />
      {data.delta && <DeltaTag mark={data.delta} />}
      <div className="tbl-head">{n.name}</div>
      <div className="tbl-cols">
        {rows.map((c) => (
          <div key={c.name} className="tbl-col">
            <span className="tbl-name">
              {c.name}
              {c.nullable ? '?' : ''}
            </span>
            <span className="tbl-type">
              {[c.primary && 'PK', c.references && 'FK', c.type].filter(Boolean).join(' ')}
            </span>
          </div>
        ))}
        {rows.length < cols.length && <div className="tbl-col tbl-more">… {cols.length - rows.length} more</div>}
      </div>
    </div>
  )
}

// The engine draws this many columns, and counts the rest (internal/render/layout.go tableRows).
const TABLE_ROWS = 14

export function ElementNode({ data, selected }: NodeProps<FlowNode<ElementData>>) {
  if (data.node.kind === 'table') return <TableNode data={data} selected={!!selected} />
  const n = data.node
  const k = kindStyle(n.kind)
  const detail = useStore(zoomSelector)
  const interp = n.description_provenance?.origin === 'model'
  const tech = n.technology || k.label

  return (
    <div
      className={`el el-${k.hue} ${n.evidence === 'referenced' ? 'el-referenced' : ''} ${selected ? 'el-selected' : ''} ${data.dim ? 'is-dim' : ''} ${data.match ? 'el-match' : ''} ${data.delta ? `el-delta-${data.delta}` : ''}`}
      title={`${n.name} · ${k.label}${n.technology ? ` · ${n.technology}` : ''}${n.files ? ` · ${n.files.length} files, ${(n.lines ?? 0).toLocaleString()} lines` : ''}`}
    >
      {/* Edges follow stored routes; these handles only satisfy the graph library. */}
      <Handle type="target" position={Position.Top} className="el-handle" isConnectable={false} />
      <Handle type="source" position={Position.Bottom} className="el-handle" isConnectable={false} />
      <div className="el-cap" />
      {data.placed && <span className="el-placed" title="Placed by a person · layout.yaml — presentation, not fact" />}
      {data.isNew && <span className="el-new">new · placed automatically</span>}
      {data.delta && <DeltaTag mark={data.delta} />}
      {data.unresolved ? (
        <span className="el-unresolved" title={`${data.unresolved} calls from here go to an address computed at run time — listed in the inspector, not drawn`}>
          ? {data.unresolved}
        </span>
      ) : null}
      {data.opens !== undefined && (
        <span className="el-opens" title={`Double-click to open its ${data.opens} ${n.kind === 'system' ? 'containers' : 'components'}`}>
          ⤵ {data.opens}
        </span>
      )}
      <div className="el-body">
        <div className="el-head">
          <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinejoin="round" strokeLinecap="round">
            <path d={k.icon} />
          </svg>
          <span className="el-name">{n.name}</span>
        </div>
        {detail && <div className="el-tech">{tech}</div>}
        {detail && (n.used_by || n.uses_many) ? (
          <div className="el-desc" title="It uses, or is used by, most of the others; those arrows are in the inspector, not on the picture">
            <span>
              {n.used_by && n.uses_many
                ? `used by ${n.used_by}, uses ${n.uses_many} of ${n.among} · not drawn`
                : n.uses_many
                  ? `uses ${n.uses_many} of ${n.among} · arrows not drawn`
                  : `used by ${n.used_by} of ${n.among} · arrows not drawn`}
            </span>
          </div>
        ) : null}

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

export function BoundaryNode({ data, draggable }: NodeProps<FlowNode<BoundaryData>>) {
  return (
    <div className={`boundary ${data.system ? 'boundary-system' : 'boundary-network'} ${draggable ? 'boundary-draggable' : ''}`}>
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
        style={e.weight && e.weight > 1 ? { strokeWidth: Math.min(4, 1.2 + Math.log2(e.weight) * 0.4) } : undefined}
      />
      {text && data.labelAt && (
        <EdgeLabelRenderer>
          <div
            className={`route-label nodrag nopan ${selected ? 'route-label-selected' : ''} ${data.dim ? 'is-dim' : ''}`}
            style={{ transform: `translate(-50%, -50%) translate(${data.labelAt.x}px, ${data.labelAt.y}px)` }}
          >
            {data.delta && <DeltaTag mark={data.delta} inline />}
            {interp && <span className="mark mark-interpreted" />}
            <span className={interp ? 'route-label-interp' : ''}>{text}</span>
          </div>
        </EdgeLabelRenderer>
      )}
    </>
  )
}

const glyphs: Record<Mark, string> = { added: '+', changed: '±', words: '±' }
const titles: Record<Mark, string> = {
  added: 'Appeared since the compared version',
  changed: 'Changed since the compared version',
  words: 'Only words changed since the compared version — a name, a description or a label',
}

/** The corner tag of a change: solid for structure, hollow for words only. */
export function DeltaTag({ mark, inline }: { mark: Mark; inline?: boolean }) {
  return (
    <span className={`delta-mark ${mark === 'words' ? 'hollow' : ''} ${inline ? 'inline' : ''}`} title={titles[mark]}>
      {glyphs[mark]}
    </span>
  )
}

/** What disappeared, ghosted at its last position: dashed, struck through, tagged −. */
export function Ghosts({ delta }: { delta: Delta }) {
  return (
    <>
      <svg className="ghost-paths" style={{ position: 'absolute', left: 0, top: 0, overflow: 'visible', pointerEvents: 'none' }} width={1} height={1}>
        {delta.ghostPaths.map((p) => (
          <path key={p.key} d={p.d} className="ghost-path">
            <title>{`${p.key.replace('>', ' → ')} disappeared since v${delta.from}`}</title>
          </path>
        ))}
      </svg>
      {delta.ghosts.map((g) => (
        <div
          key={g.id}
          className="ghost"
          style={{ position: 'absolute', left: g.rect.x, top: g.rect.y, width: g.rect.w, height: g.rect.h }}
          title={`${g.name} disappeared since v${delta.from}`}
        >
          <span className="delta-mark">−</span>
          <div className="ghost-name">{g.name}</div>
          {g.tech && <div className="ghost-tech">{g.tech}</div>}
        </div>
      ))}
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
