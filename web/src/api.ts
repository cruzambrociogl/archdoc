// The shapes `archdoc serve` returns, and the one way the app fetches them. The app computes
// nothing of its own: every value on screen is one of these, as the engine stored it.

import { createContext, useContext, useEffect, useState } from 'react'

export type Origin = 'extraction' | 'catalog' | 'rules' | 'model'

export interface Provenance {
  origin?: Origin
  file: string
  line: number
  column?: number
  note?: string
}

export interface Node {
  id: string
  name: string
  name_provenance?: Provenance
  kind: string
  description?: string
  description_provenance?: Provenance
  technology?: string
  technology_provenance?: Provenance
  evidence: 'declared' | 'referenced'
  parent?: string
  networks?: string[] | null
  /** The application's own code, and what ties it to this element (a manifest, a build line). */
  dir?: string
  dir_provenance?: Provenance
  /** A component's files, in path order, and their total lines. */
  files?: string[]
  lines?: number
  /** A table's columns, as the code declares them. */
  columns?: Column[]
  provenance: Provenance
}

export interface Column {
  name: string
  type?: string
  primary?: boolean
  nullable?: boolean
  /** The table it references, by element ID. */
  references?: string
  provenance: Provenance
}

export interface Edge {
  from: string
  to: string
  label?: string
  label_provenance?: Provenance
  technology?: string
  traffic?: boolean
  provenance: Provenance[] | null
  /** How many imports a component edge stands for; its provenance cites one per importing file, at most ten. */
  weight?: number
}

export interface Model {
  name: string
  source: string
  nodes: Node[] | null
  edges: Edge[] | null
  entries?: Entry[]
  unresolved?: Unresolved[]
  flows?: Flow[]
  explanations?: Explanation[]
}

/** What a model wrote about an element, every sentence citing the lines it rests on (F-19, F-36). */
export interface Explanation {
  element: string
  claims: { text: string; facts: string[]; cites: Provenance[] }[]
  fingerprint: string
  /** The element's facts changed since this was written; it is the last answer, not a current one. */
  stale?: boolean
  provenance: Provenance
}

/** What an entry sets off, followed through the code (F-12). */
export interface Flow {
  entry: string
  participants: { id: string; name: string; kind: 'class' | 'table' | 'unresolved'; component?: string; element?: string }[]
  steps: { from: string; to: string; call: string; depth: number; note?: string; provenance: Provenance }[]
  cut?: boolean
}

export interface Summary {
  name: string
  root: string
  source: string
  commit: string
  latest_version: number
  versions: number
  runs: number
  frontend_built: boolean
}

export interface Version {
  id: number
  created_at: string
  commit: string
  source: string
}

export interface ModelResponse {
  version: number
  created_at: string
  commit: string
  model: Model
  context: Model
  container: Model
  /** The containers that have a component view. */
  components: string[]
}

/** A container whose code was read: it opens onto its components. */
export interface Opening {
  id: string
  name: string
  components: number
}

/** The component view of a container is the level `component:<container id>`; its data view `data:<container id>`. */
export const componentLevel = (id: string) => `component:${id}`
export const dataLevel = (id: string) => `data:${id}`
export const componentOf = (level?: string) => (level?.startsWith('component:') ? level.slice('component:'.length) : undefined)
export const dataOf = (level?: string) => (level?.startsWith('data:') ? level.slice('data:'.length) : undefined)
/** The container a level is inside of, whichever lens. */
export const insideOf = (level?: string) => componentOf(level) ?? dataOf(level)

// The layout a view is drawn at (internal/archdoc/layout.go), and the scene that pairs them.
export interface Point {
  x: number
  y: number
}

export interface Rect {
  x: number
  y: number
  w: number
  h: number
}

export interface Layout {
  version: number
  width: number
  height: number
  boxes: { id: string; rect: Rect }[] | null
  groups?: { name: string; label?: string; internal?: boolean; system?: boolean; rect: Rect; label_at: Point }[] | null
  paths?: { from: string; to: string; curve: Point[]; tip?: Point; label_at?: Point }[] | null
}

export interface SceneResponse {
  version: number
  /** "context", "container", or "component:<container id>". */
  view: string
  model: Model
  layout: Layout
  components: Opening[]
  /** Containers whose code declares tables; `components` here counts their tables. */
  data: Opening[]
  /** How many elements the container view holds: what the system box opens onto. */
  containers: number
  /** How layout.yaml met this view: what a person placed, what is new since, what names nothing. */
  arrangement: { file: string; hash: string; placed: string[]; new: string[]; stale: string[] }
}

export interface CoverageResponse {
  source: string
  read: { file: string; used: boolean; why: string }[]
  known: { label: string; value: string }[]
  complete: number
  items: number
  gaps: { rule: string; gaps: { rule: string; element: string; message: string }[] }[]
  unconnected: string[]
  limits: { limit: string; needed: string }[]
  /** Each running application's code: read, with how its imports resolved, or not read yet. */
  code?: CodeRead[]
  /** Calls whose target is computed at run time. */
  unresolved?: Unresolved[]
  /** How many routes and pages the code declares. */
  routes?: number
  pages?: number
  tables?: number
  tables_in?: string[]
}

/** A way into the system its code declares — an HTTP route (F-04, F-13). */
export interface Entry {
  id: string
  kind: string
  method: string
  path: string
  handler: string
  summary?: string
  summary_provenance?: Provenance
  container: string
  component?: string
  uses?: { name: string; component?: string; how: string; provenance: Provenance }[]
  prefix_provenance?: Provenance
  path_note?: string
  provenance: Provenance
}

/** Something the code does that could not be tied to an element (D-6). */
export interface Unresolved {
  container: string
  component?: string
  what: string
  note: string
  provenance: Provenance
}

export interface CodeRead {
  app: string
  container: string
  language: string
  read: boolean
  root?: string
  files: number
  lines: number
  components: number
  imports: Partial<Record<'path' | 'alias' | 'module' | 'package' | 'unresolved', number>>
  unresolved: { spec: string; provenance: Provenance }[]
  partial: string[]
}

export interface SavedView {
  name: string
  level: string
  focus?: string
  find?: string
  dim?: boolean
}

export interface ViewsResponse {
  file: string
  hash: string
  views: SavedView[]
}

export interface Change {
  element: string
  field: string
  before: string
  after: string
}

export interface DiffResponse {
  from: number
  to: number
  structural: boolean
  empty: boolean
  diff: {
    added_nodes: Node[] | null
    removed_nodes: Node[] | null
    added_edges: Edge[] | null
    removed_edges: Edge[] | null
    changed: Change[] | null
  }
}

export interface Run {
  id: number
  started_at: string
  finished_at: string
  status: string
  commit: string
  egress_mode: string
  model: string
  requests: number
  bytes_sent: number
  tokens_in: number
  tokens_out: number
  cost_usd: number
  cost_known: boolean
}

export interface Exchange {
  method: string
  url: string
  status: number
  body: string
}

export interface Op {
  kind: string
  target: string
  to?: string
  value?: string
  origin: Origin
  provenance: Provenance
}

export interface RulesResponse {
  file: string
  exists: boolean
  /** The rules were read from the old location at the repository root. */
  legacy?: boolean
  /** Both locations exist; the root file is ignored. */
  shadowed?: boolean
  rules: {
    line: number
    set: Record<string, string> | null
    exclude: boolean
    remove: boolean
    match: { name: string; image: string; kind: string }
    edge?: { from: string; to: string }
  }[]
  operations: Op[] | null
  /** rule is the check that raised it (RUL-05); element is where — for a rule, "rules.yaml:6". */
  findings: { rule: string; severity: string; element: string; message: string }[]
}

export interface DocsResponse {
  dir: string
  generated: string[] | null
  human: { number: number; title: string; file: string }[] | null
}

export interface CompletenessResponse {
  dir: string
  architecture_changed: string
  sections: {
    number: number
    title: string
    file: string
    state: 'missing' | 'not started' | 'written' | 'may be stale'
    modified?: string
  }[]
}

// ——— live or published ———
// The same app runs in two modes (surface-spec §3). Live, it reads archdoc serve's API. Published —
// built by 'archdoc export --site' — it reads the same responses from static files beside it; the
// marker is written into index.html by the exporter.

export const published = !!document.querySelector('meta[name="archdoc-mode"][content="published"]')

/** The file a published site reads an API path from. Mirrors serve.StaticName in Go. */
export function staticName(apiPath: string): string {
  const [p, query = ''] = apiPath.replace(/#.*$/, '').replace(/^\/api\//, '').split('?')
  if (p.startsWith('docs/')) return `data/${p}`
  const ext = p === 'svg' ? '.svg' : '.json'
  const q = new URLSearchParams(query)
  const keys = [...new Set([...q.keys()])].sort()
  // A view name may hold an element ID ("component:app:packages/cli"): no slash or colon in a file name.
  const safe = (v: string | null) => (v ?? '').replace(/\//g, '_').replace(/:/g, '~')
  const tail = keys.length ? '@' + keys.map((k) => `${k}=${safe(q.get(k))}`).join(',') : ''
  return `data/${p.replace(/\//g, '_')}${tail}${ext}`
}

/** Where a path is fetched from in this mode. */
export const source = (path: string) => (published && path.startsWith('/api/') ? staticName(path) : path)

async function request(path: string): Promise<Response> {
  const r = await fetch(source(path))
  if (!r.ok) {
    let msg = r.statusText
    try {
      msg = (await r.json()).error ?? msg
    } catch {
      /* not JSON: keep the status text */
    }
    throw new Error(msg)
  }
  return r
}

/** Fetches path whenever it changes; null fetches nothing. Stale answers are never shown. */
export function useApi<T>(path: string | null, as: 'json' | 'text' = 'json') {
  // A path that differs only by a trailing `#n` is fetched again: that is how a screen reloads.
  const [state, setState] = useState<{ path?: string; data?: T; error?: string }>({})
  useEffect(() => {
    if (!path) return
    let live = true
    request(path)
      .then((r) => (as === 'json' ? r.json() : r.text()))
      .then(
        (data) => live && setState({ path, data: data as T }),
        (e: Error) => live && setState({ path, error: e.message }),
      )
    return () => {
      live = false
    }
  }, [path, as])
  const current = state.path === path
  return {
    data: current ? state.data : undefined,
    error: current ? state.error : undefined,
    loading: !!path && !current,
  }
}

// ——— actions ———
// Anything that runs or writes goes through action(): a POST carrying this session's token, which
// the server checks together with the Origin (internal/serve/guard.go, surface-spec §11.5). The
// token is fetched once per page from /api/session, which answers same-origin pages only.

let session: Promise<string> | null = null

function token(): Promise<string> {
  session ??= request('/api/session')
    .then((r) => r.json())
    .then((s: { token: string }) => s.token)
    .catch((e) => {
      session = null
      throw e
    })
  return session
}

export async function action<T>(path: string, body?: unknown): Promise<T> {
  const r = await fetch(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-Archdoc-Token': await token() },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  const text = await r.text()
  let data: unknown = undefined
  try {
    data = text ? JSON.parse(text) : undefined
  } catch {
    /* not JSON */
  }
  if (!r.ok) throw new Error((data as { error?: string } | undefined)?.error ?? r.statusText)
  return data as T
}

/**
 * Where citations open. Live: the repository's absolute path, so a citation opens in the editor.
 * Published: the repository host at the commit, when a remote is known; with no known remote a
 * citation is plain text, never a guessed link (surface-spec S-3).
 */
export interface Place {
  root: string
  remote?: string
  commit?: string
}
export const RootContext = createContext<Place>({ root: '' })

export function editorLink(place: Place | string, file: string, line?: number): string | undefined {
  const p = typeof place === 'string' ? { root: place } : place
  if (published) {
    if (!p.remote || !p.commit) return undefined
    return `${p.remote}/blob/${p.commit}/${file.replace(/^\.\//, '')}${line ? `#L${line}` : ''}`
  }
  const path = file.startsWith('/') ? file : `${p.root}/${file}`
  return `vscode://file${path}${line ? `:${line}` : ''}`
}

export function useEditorLink() {
  const place = useContext(RootContext)
  return (file: string, line?: number) => editorLink(place, file, line)
}

export interface Session {
  mode: 'local' | 'published'
  remote?: string
  commit?: string
  generated_at?: string
  version?: number
  baseline?: number
}

export function when(iso?: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleString()
}

export function bytes(n: number): string {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / 1024 / 1024).toFixed(2)} MB`
}
