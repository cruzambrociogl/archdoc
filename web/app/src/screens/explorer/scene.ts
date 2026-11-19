// Turns the engine's scene — a view and the layout it is drawn at — into React Flow nodes and
// edges. Nothing is laid out here: every position and every route comes from the engine, so the
// app draws the same arrangement as the committed SVG (surface-spec §10.1).

import type { Edge as FlowEdge, Node as FlowNode } from '@xyflow/react'
import type { Change, DiffResponse, Edge, Layout, Node, Point, Rect, SceneResponse } from '../../api'

export interface ElementData extends Record<string, unknown> {
  node: Node
  dim: boolean
  match: boolean
  /** Moved by a person and not saved yet. Presentation, not truth. */
  placed: boolean
  /** Added since the arrangement was saved, so the engine placed it. */
  isNew: boolean
  /** Its change against the compared version, if one is set. */
  delta?: Mark
  /** A container whose code was read: how many components it opens onto. */
  opens?: number
  /** Calls leaving this box for an address computed at run time: shown as a count, never as a target. */
  unresolved?: number
  /** A word the highlight puts on the box: "through immich-server", "Partly read", "Written by AI". */
  tag?: string
  /** What the tag means, and so its colour: the ink of the drawing, grey for what was not read, the AI's blue. */
  tone?: Tone
  /** Hatched in grey: not read, or read only in part. */
  hatch?: boolean
}

export type Tone = 'ink' | 'muted' | 'interp'

/**
 * The highlight (platform-surface §17.5): one thing shown on the system. What belongs to it stays
 * lit, the rest dims to about a fifth, and a box can carry a small tag. Every view that shows
 * something on a diagram — a part found by name, a feature's parts, a change, one part's reach —
 * passes one.
 */
export interface Highlight {
  lit: Set<string>
  tags?: Map<string, string>
  /** One meaning for every tag of this view; ink when none is said. */
  tone?: Tone
  /** Boxes hatched in grey (rule: the hatch is only for Not read and Partly read, and Down). */
  hatched?: Set<string>
}

/** Find a part: the boxes whose name, id or technology holds what was typed; none when nothing was. */
export function findParts(nodes: Node[], typed: string): Highlight | undefined {
  const q = typed.trim().toLowerCase()
  if (!q) return undefined
  const hit = (n: Node) => n.name.toLowerCase().includes(q) || n.id.toLowerCase().includes(q) || (n.technology ?? '').toLowerCase().includes(q)
  return { lit: new Set(nodes.filter(hit).map((n) => n.id)) }
}

export interface BoundaryData extends Record<string, unknown> {
  label: string
  system: boolean
  internal: boolean
  /** A column of the data-flow view, not a boundary. */
  stage: boolean
}

export interface RouteData extends Record<string, unknown> {
  edge: Edge
  d: string
  labelAt?: Point
  dim: boolean
  delta?: Mark
}

// ——— the change overlay (surface-spec §5.2, §6.7) ———
// Change is stroke weight and a corner tag, never a third hue, and never the truth-state forms.
// Structural change (appeared, rewired, a technology or kind changed) takes a solid tag; a change
// of words only (a name, a description, a label) takes a hollow one, so a relabel never reads as a
// rewiring. What disappeared is ghosted at its last position.

export type Mark = 'added' | 'changed' | 'words'

const wordFields = new Set(['name', 'description', 'label'])

export interface Delta {
  from: number
  to: number
  nodes: Map<string, Mark>
  edges: Map<string, Mark>
  changes: Map<string, Change[]>
  ghosts: { id: string; name: string; tech?: string; rect: Rect }[]
  ghostPaths: { key: string; d: string }[]
  count: { added: number; changed: number; removed: number; words: number }
}

/** The diff between two versions, as marks on this scene, and ghosts from the compared one. */
export function toDelta(diff: DiffResponse, before: SceneResponse | undefined): Delta {
  const x = diff.diff
  const nodes = new Map<string, Mark>()
  const edges = new Map<string, Mark>()
  const changes = new Map<string, Change[]>()

  for (const n of x.added_nodes ?? []) nodes.set(n.id, 'added')
  for (const e of x.added_edges ?? []) edges.set(edgeKey(e), 'added')
  for (const c of x.changed ?? []) {
    const id = c.element.includes(' → ') ? c.element.replace(' → ', '>') : c.element
    changes.set(id, [...(changes.get(id) ?? []), c])
    const isEdge = id.includes('>')
    const map = isEdge ? edges : nodes
    const mark: Mark = wordFields.has(c.field) ? 'words' : 'changed'
    if (map.get(id) !== 'added' && map.get(id) !== 'changed') map.set(id, mark)
  }

  const oldBoxes = new Map((before?.layout.boxes ?? []).map((b) => [b.id, b.rect]))
  const oldPaths = new Map((before?.layout.paths ?? []).map((p) => [edgeKey(p), p]))
  const ghosts = (x.removed_nodes ?? [])
    .filter((n) => oldBoxes.has(n.id))
    .map((n) => ({ id: n.id, name: n.name, tech: n.technology, rect: oldBoxes.get(n.id)! }))
  const ghostPaths = (x.removed_edges ?? [])
    .map((e) => oldPaths.get(edgeKey(e)))
    .filter((p): p is NonNullable<typeof p> => !!p)
    .map((p) => ({ key: edgeKey(p), d: curvePath(p.curve, p.tip) }))

  const marks = [...nodes.values(), ...edges.values()]
  return {
    from: diff.from,
    to: diff.to,
    nodes,
    edges,
    changes,
    ghosts,
    ghostPaths,
    count: {
      added: marks.filter((m) => m === 'added').length,
      changed: marks.filter((m) => m === 'changed').length,
      removed: (x.removed_nodes ?? []).length + (x.removed_edges ?? []).length,
      words: marks.filter((m) => m === 'words').length,
    },
  }
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
 * what does not match the search; `highlight` dims what a view does not show. Dimming is
 * presentation only — nothing is hidden or moved.
 */
export function toFlow(
  sc: SceneResponse,
  layout: Layout,
  opts: { selected: string | null; focus: boolean; find: string; placed: Set<string>; isNew: Set<string>; delta?: Delta; opens?: Map<string, number>; highlight?: Highlight },
) {
  const nodes = sc.model.nodes ?? []
  const edges = sc.model.edges ?? []
  const byId = new Map(nodes.map((n) => [n.id, n]))
  const found = findParts(nodes, opts.find)
  const shown = opts.highlight
  const loose = new Map<string, number>()
  for (const u of sc.unresolved ?? []) for (const id of [u.container, u.component]) if (id) loose.set(id, (loose.get(id) ?? 0) + 1)

  // What stays lit when focusing: the selection and its direct neighbours.
  // A component view is dense — every import between two directories is an arrow — so selecting a
  // component focuses on it without being asked.
  const focus = opts.focus || /^(component|data|structure):/.test(sc.view)
  const near = new Set<string>()
  if (focus && opts.selected && byId.has(opts.selected)) {
    near.add(opts.selected)
    for (const e of edges) {
      if (e.from === opts.selected) near.add(e.to)
      if (e.to === opts.selected) near.add(e.from)
    }
  }
  const dimNode = (n: Node) => (near.size > 0 && !near.has(n.id)) || (!!found && !found.lit.has(n.id)) || (!!shown && !shown.lit.has(n.id))

  const flowNodes: FlowNode[] = []

  // Boundaries first and underneath; largest first, so a network sits above the system around it.
  // A boundary is picked up by its label only: the system's covers nearly the whole canvas, and a
  // body that caught the pointer would turn every drag across it into a move instead of a pan.
  const groups = [...(layout.groups ?? [])].sort((a, b) => b.rect.w * b.rect.h - a.rect.w * a.rect.h)
  groups.forEach((g, i) => {
    flowNodes.push({
      id: `boundary:${g.name}`,
      type: 'boundary',
      position: { x: g.rect.x, y: g.rect.y },
      width: g.rect.w,
      height: g.rect.h,
      data: { label: g.label || g.name, system: !!g.system, internal: !!g.internal, stage: !!g.stage } satisfies BoundaryData,
      selectable: false,
      dragHandle: '.boundary-label',
      style: { pointerEvents: 'none' },
      focusable: false,
      zIndex: -100 + i,
    })
  })

  for (const b of layout.boxes ?? []) {
    const n = byId.get(b.id)
    if (!n) continue
    flowNodes.push({
      id: n.id,
      type: 'element',
      position: { x: b.rect.x, y: b.rect.y },
      width: b.rect.w,
      height: b.rect.h,
      data: { node: n, dim: dimNode(n), match: !!found?.lit.has(n.id), placed: opts.placed.has(n.id), isNew: opts.isNew.has(n.id), delta: opts.delta?.nodes.get(n.id), opens: opts.opens?.get(n.id), unresolved: loose.get(n.id), tag: shown?.tags?.get(n.id), tone: shown?.tone, hatch: shown?.hatched?.has(n.id) } satisfies ElementData,
      selected: opts.selected === n.id,
    })
  }

  const paths = new Map((layout.paths ?? []).map((p) => [edgeKey(p), p]))
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
      data: { edge: e, d: curvePath(p.curve, p.tip), labelAt: p.label_at, dim:
          !lit ||
          (!!found && !found.lit.has(e.from) && !found.lit.has(e.to)) ||
          (!!shown && !(shown.lit.has(e.from) && shown.lit.has(e.to))), delta: opts.delta?.edges.get(key) } satisfies RouteData,
      selected: opts.selected === key,
    })
  }

  return { nodes: flowNodes, edges: flowEdges }
}

// ——— a draft arrangement, previewed ———
// While a person drags, the app previews what the engine will draw once the arrangement is saved.
// This is the engine's arrange.Apply (internal/arrange/arrange.go), ported: boxes move, boundaries
// regrow around what they contained, relationships whose ends moved together keep their curve,
// and the others are drawn straight between the boxes. After saving, the engine's own result
// replaces the preview.


export type Draft = Record<string, Point>

const ARROW = 8

export function arrangeDraft(l: Layout, draft: Draft): Layout {
  const ids = Object.keys(draft)
  if (!ids.length) return l
  const boxes = l.boxes ?? []
  const before = new Map(boxes.map((b) => [b.id, b.rect]))
  const after = new Map(boxes.map((b) => [b.id, draft[b.id] ? { ...b.rect, x: Math.max(0, draft[b.id].x), y: Math.max(0, draft[b.id].y) } : b.rect]))

  const inside = (r: Rect, o: Rect) => r.x >= o.x && r.y >= o.y && r.x + r.w <= o.x + o.w && r.y + r.h <= o.y + o.h
  const bounds = (members: string[], m: Map<string, Rect>): Rect => {
    let x0 = Infinity, y0 = Infinity, x1 = -Infinity, y1 = -Infinity
    for (const id of members) {
      const r = m.get(id)!
      x0 = Math.min(x0, r.x); y0 = Math.min(y0, r.y); x1 = Math.max(x1, r.x + r.w); y1 = Math.max(y1, r.y + r.h)
    }
    return { x: x0, y: y0, w: x1 - x0, h: y1 - y0 }
  }

  const groups = (l.groups ?? []).map((g) => {
    const members = boxes.filter((b) => inside(b.rect, g.rect)).map((b) => b.id)
    if (!members.length || !members.some((id) => before.get(id) !== after.get(id))) return g
    const was = bounds(members, before), now = bounds(members, after)
    const rect = { x: now.x - (was.x - g.rect.x), y: now.y - (was.y - g.rect.y), w: now.w + (g.rect.w - was.w), h: now.h + (g.rect.h - was.h) }
    return { ...g, rect, label_at: { x: g.label_at.x + rect.x - g.rect.x, y: g.label_at.y + rect.y - g.rect.y } }
  })

  const delta = (id: string): Point => {
    const b = before.get(id)!, a = after.get(id)!
    return { x: a.x - b.x, y: a.y - b.y }
  }
  const paths = (l.paths ?? []).map((p) => {
    const df = delta(p.from), dt = delta(p.to)
    if (!df.x && !df.y && !dt.x && !dt.y) return p
    if (df.x === dt.x && df.y === dt.y) {
      const mv = (q: Point) => ({ x: q.x + df.x, y: q.y + df.y })
      return { ...p, curve: p.curve.map(mv), tip: p.tip && mv(p.tip), label_at: p.label_at && mv(p.label_at) }
    }
    const f = after.get(p.from)!, t = after.get(p.to)!
    const c1 = { x: f.x + f.w / 2, y: f.y + f.h / 2 }, c2 = { x: t.x + t.w / 2, y: t.y + t.h / 2 }
    const start = clip(c1, c2, f), tip = clip(c2, c1, t)
    const dx = tip.x - start.x, dy = tip.y - start.y, len = Math.hypot(dx, dy)
    const end = len > ARROW ? { x: tip.x - (dx / len) * ARROW, y: tip.y - (dy / len) * ARROW } : tip
    const at = (k: number) => ({ x: start.x + (end.x - start.x) * k, y: start.y + (end.y - start.y) * k })
    return {
      ...p,
      curve: [start, at(1 / 3), at(2 / 3), end],
      tip: p.tip ? tip : undefined,
      label_at: p.label_at ? { x: (start.x + tip.x) / 2, y: (start.y + tip.y) / 2 } : undefined,
    }
  })

  return { ...l, boxes: boxes.map((b) => ({ id: b.id, rect: after.get(b.id)! })), groups, paths }
}

function clip(c: Point, toward: Point, r: Rect): Point {
  const dx = toward.x - c.x, dy = toward.y - c.y
  if (!dx && !dy) return c
  const sx = dx ? r.w / 2 / Math.abs(dx) : Infinity
  const sy = dy ? r.h / 2 / Math.abs(dy) : Infinity
  const s = Math.min(sx, sy)
  return { x: c.x + dx * s, y: c.y + dy * s }
}

/** The boxes a boundary contained as the engine drew it — what moves when the boundary is dragged. */
export function membersOf(l: Layout, boundary: string): string[] {
  const g = (l.groups ?? []).find((x) => x.name === boundary)
  if (!g) return []
  return (l.boxes ?? [])
    .filter((b) => b.rect.x >= g.rect.x && b.rect.y >= g.rect.y && b.rect.x + b.rect.w <= g.rect.x + g.rect.w && b.rect.y + b.rect.h <= g.rect.y + g.rect.h)
    .map((b) => b.id)
}

// ——— reshaping a boundary ———
// A boundary has no size of its own: it is drawn around what it holds. Pulling its edge is a way to
// move many boxes at once — what it holds is laid out again in rows as wide as the edge was pulled
// (or, pulled by its top or bottom, in columns as tall), and what is saved is only where each box
// now sits, like any other move. A boundary inside it moves as one piece, with its boxes.

/** Space between boxes laid out by a reshape: the engine's own, Graphviz's nodesep and ranksep. */
const GAP = { x: 44, y: 64 }

/** What a boundary held when the reshape began, and where. */
export interface Reshape {
  /** Each piece laid out: a box, or a boundary inside with the boxes it holds. */
  items: { ids: string[]; rect: Rect }[]
  /** The space between the boundary's edge and what it holds, kept as it is reshaped. */
  pad: { left: number; top: number; right: number; bottom: number }
}

/**
 * What the boundary `name` holds, as the engine drew it (`base`), at the positions shown now
 * (`current`, the draft applied). Undefined when it holds nothing.
 */
export function reshapeStart(base: Layout, current: Layout, name: string): Reshape | undefined {
  const groups = base.groups ?? []
  const g = groups.find((x) => x.name === name)
  if (!g) return undefined
  const inside = (r: Rect, o: Rect) => r.x >= o.x && r.y >= o.y && r.x + r.w <= o.x + o.w && r.y + r.h <= o.y + o.h
  const same = (a: Rect, b: Rect) => a.x === b.x && a.y === b.y && a.w === b.w && a.h === b.h
  const boxes = base.boxes ?? []
  const now = new Map((current.boxes ?? []).map((b) => [b.id, b.rect]))
  const nowGroup = new Map((current.groups ?? []).map((x) => [x.name, x.rect]))

  // The boundaries directly inside this one: not inside another that is.
  const inner = groups.filter((x) => x.name !== name && !same(x.rect, g.rect) && inside(x.rect, g.rect))
  const top = inner.filter((x) => !inner.some((y) => y !== x && !same(y.rect, x.rect) && inside(x.rect, y.rect)))

  const items: Reshape['items'] = []
  const taken = new Set<string>()
  for (const x of top) {
    const ids = boxes.filter((b) => inside(b.rect, x.rect) && !taken.has(b.id)).map((b) => b.id)
    if (!ids.length) continue
    ids.forEach((id) => taken.add(id))
    items.push({ ids, rect: nowGroup.get(x.name) ?? x.rect })
  }
  for (const b of boxes) {
    if (taken.has(b.id) || !inside(b.rect, g.rect)) continue
    items.push({ ids: [b.id], rect: now.get(b.id) ?? b.rect })
  }
  if (!items.length) return undefined

  const at = nowGroup.get(name) ?? g.rect
  let x0 = Infinity, y0 = Infinity, x1 = -Infinity, y1 = -Infinity
  for (const { rect: r } of items) {
    x0 = Math.min(x0, r.x); y0 = Math.min(y0, r.y); x1 = Math.max(x1, r.x + r.w); y1 = Math.max(y1, r.y + r.h)
  }
  return { items, pad: { left: x0 - at.x, top: y0 - at.y, right: at.x + at.w - x1, bottom: at.y + at.h - y1 } }
}

/**
 * Where each box goes for the boundary to take the shape `to`. `along` says which edge was pulled:
 * a side lays out rows that fit the width, the top or bottom columns that fit the height. Pieces
 * keep the order they were read in — row by row, or column by column — so what sat together stays
 * together. A piece wider (or taller) than the shape asked gets a row (or column) of its own.
 */
export function reshape(s: Reshape, base: Layout, current: Layout, to: Rect, along: 'width' | 'height'): Draft {
  const now = new Map((current.boxes ?? []).map((b) => [b.id, b.rect]))
  const origin = { x: to.x + s.pad.left, y: to.y + s.pad.top }
  const rows = along === 'width'
  // Boxes the engine put in one rank share a centre line; read them in bands of 20 points, as the
  // engine does when it folds a rank.
  const band = (v: number) => Math.round(v / 20)
  const centre = (r: Rect) => ({ x: r.x + r.w / 2, y: r.y + r.h / 2 })
  const order = [...s.items].sort((a, b) => {
    const p = centre(a.rect), q = centre(b.rect)
    return rows ? band(p.y) - band(q.y) || p.x - q.x : band(p.x) - band(q.x) || p.y - q.y
  })

  const draft: Draft = {}
  const room = rows ? to.w - s.pad.left - s.pad.right : to.h - s.pad.top - s.pad.bottom
  let along0 = 0, across = 0, line = 0
  for (const it of order) {
    const len = rows ? it.rect.w : it.rect.h
    const thick = rows ? it.rect.h : it.rect.w
    if (along0 > 0 && along0 + len > room) {
      across += line + (rows ? GAP.y : GAP.x)
      along0 = 0
      line = 0
    }
    const x = origin.x + (rows ? along0 : across)
    const y = origin.y + (rows ? across : along0)
    const dx = x - it.rect.x, dy = y - it.rect.y
    for (const id of it.ids) {
      const r = now.get(id) ?? base.boxes?.find((b) => b.id === id)?.rect
      if (r) draft[id] = { x: Math.round(Math.max(0, r.x + dx)), y: Math.round(Math.max(0, r.y + dy)) }
    }
    along0 += len + (rows ? GAP.x : GAP.y)
    line = Math.max(line, thick)
  }
  return draft
}
