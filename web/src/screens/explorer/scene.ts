// Turns the engine's scene — a view and the layout it is drawn at — into React Flow nodes and
// edges. Nothing is laid out here: every position and every route comes from the engine, so the
// app draws the same arrangement as the committed SVG (surface-spec §10.1).

import type { Edge as FlowEdge, Node as FlowNode } from '@xyflow/react'
import type { Edge, Node, Point, SceneResponse } from '../../api'

export interface ElementData extends Record<string, unknown> {
  node: Node
  dim: boolean
  match: boolean
}

export interface BoundaryData extends Record<string, unknown> {
  label: string
  system: boolean
  internal: boolean
}

export interface RouteData extends Record<string, unknown> {
  edge: Edge
  d: string
  labelAt?: Point
  dim: boolean
}

/** A Graphviz Bézier chain — a start point, then groups of three — as an SVG path, ending at the tip. */
export function curvePath(curve: Point[], tip?: Point): string {
  if (!curve.length) return ''
  const p = (q: Point) => `${q.x.toFixed(1)},${q.y.toFixed(1)}`
  let d = `M${p(curve[0])}`
  for (let i = 1; i + 2 < curve.length; i += 3) d += ` C${p(curve[i])} ${p(curve[i + 1])} ${p(curve[i + 2])}`
  if (tip) d += ` L${p(tip)}`
  return d
}

export const edgeKey = (e: { from: string; to: string }) => `${e.from}>${e.to}`

/**
 * The flow graph for a scene. `focus` dims what is not one hop from the selection; `find` dims
 * what does not match the search. Dimming is presentation only — nothing is hidden or moved.
 */
export function toFlow(sc: SceneResponse, opts: { selected: string | null; focus: boolean; find: string }) {
  const nodes = sc.model.nodes ?? []
  const edges = sc.model.edges ?? []
  const byId = new Map(nodes.map((n) => [n.id, n]))
  const q = opts.find.trim().toLowerCase()

  // What stays lit when focusing: the selection and its direct neighbours.
  const near = new Set<string>()
  if (opts.focus && opts.selected && byId.has(opts.selected)) {
    near.add(opts.selected)
    for (const e of edges) {
      if (e.from === opts.selected) near.add(e.to)
      if (e.to === opts.selected) near.add(e.from)
    }
  }
  const matches = (n: Node) =>
    !q || n.name.toLowerCase().includes(q) || n.id.toLowerCase().includes(q) || (n.technology ?? '').toLowerCase().includes(q)
  const dimNode = (n: Node) => (near.size > 0 && !near.has(n.id)) || (!!q && !matches(n))

  const flowNodes: FlowNode[] = []

  // Boundaries first and underneath; largest first, so a network sits above the system around it.
  const groups = [...(sc.layout.groups ?? [])].sort((a, b) => b.rect.w * b.rect.h - a.rect.w * a.rect.h)
  groups.forEach((g, i) => {
    flowNodes.push({
      id: `boundary:${g.name}`,
      type: 'boundary',
      position: { x: g.rect.x, y: g.rect.y },
      width: g.rect.w,
      height: g.rect.h,
      data: { label: g.label || g.name, system: !!g.system, internal: !!g.internal } satisfies BoundaryData,
      selectable: false,
      draggable: false,
      focusable: false,
      zIndex: -100 + i,
    })
  })

  for (const b of sc.layout.boxes ?? []) {
    const n = byId.get(b.id)
    if (!n) continue
    flowNodes.push({
      id: n.id,
      type: 'element',
      position: { x: b.rect.x, y: b.rect.y },
      width: b.rect.w,
      height: b.rect.h,
      data: { node: n, dim: dimNode(n), match: !!q && matches(n) } satisfies ElementData,
      selected: opts.selected === n.id,
      draggable: false,
    })
  }

  const paths = new Map((sc.layout.paths ?? []).map((p) => [edgeKey(p), p]))
  const flowEdges: FlowEdge[] = []
  for (const e of edges) {
    const p = paths.get(edgeKey(e))
    if (!p || !byId.has(e.from) || !byId.has(e.to)) continue
    const key = edgeKey(e)
    const lit = near.size === 0 || (near.has(e.from) && near.has(e.to) && (e.from === opts.selected || e.to === opts.selected))
    flowEdges.push({
      id: key,
      source: e.from,
      target: e.to,
      type: 'routed',
      data: { edge: e, d: curvePath(p.curve, p.tip), labelAt: p.label_at, dim: !lit || (!!q && dimNode(byId.get(e.from)!) && dimNode(byId.get(e.to)!)) } satisfies RouteData,
      selected: opts.selected === key,
    })
  }

  return { nodes: flowNodes, edges: flowEdges }
}
