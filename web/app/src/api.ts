// The shapes `archdoc serve` returns, and the one way the app fetches them. The app computes
// nothing of its own: every value on screen is one of these, as the engine stored it.

import { createContext, useContext, useEffect, useState } from 'react'
import type { Capabilities, Place } from './source'
import { useSource } from './source'
import { reached, unanswered, unreachable, useConnection } from './connection'

/** What nothing answering means, in each place. */
export const notAnswering: Record<Capabilities['place'], string> = {
  local: 'archdoc serve isn’t answering',
  published: 'this site isn’t answering',
  hosted: 'archdoc isn’t answering',
}

export type { Place }

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
  /** In a component view: most of the others use it, so its box says so and no arrow is drawn into it. */
  used_by?: number
  among?: number
  /** The mirror: it uses most of the others, so no arrow is drawn out of it. */
  uses_many?: number
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
  dependencies?: Package[]
  /** External systems a context or container view leaves out: the ones the fewest containers reach. */
  folded?: number
}

/** A package a container's manifest declares, and where its code imports it (F-14). */
export interface Package {
  container: string
  name: string
  version?: string
  dev?: boolean
  imports: number
  components?: string[]
  provenance: Provenance
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
  participants: { id: string; name: string; kind: 'class' | 'module' | 'table' | 'job' | 'unresolved'; component?: string; element?: string }[]
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
  /** A project small enough for one page: told as a story, shown without the levels a system needs. */
  tiny?: boolean
  story?: { text: string; provenance?: Provenance; interpreted?: boolean }[]
  /** What the navigation's explorer card says: containers, components, tables, flows, ways in, unresolved calls. */
  counts?: Partial<Record<'containers' | 'components' | 'tables' | 'flows' | 'entries' | 'unresolved', number>>
  /** Each container's kind, in model order: what the card draws. */
  kinds?: string[]
  /** The system has a data-flow view: its containers by stage, from who puts data in to where it is kept. */
  dataflow?: boolean
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
/** A container's code by folder, where its components are features: `structure:<container id>`. */
export const structureLevel = (id: string) => `structure:${id}`
export const structureOf = (level?: string) => (level?.startsWith('structure:') ? level.slice('structure:'.length) : undefined)
export const componentOf = (level?: string) => (level?.startsWith('component:') ? level.slice('component:'.length) : undefined)
export const dataOf = (level?: string) => (level?.startsWith('data:') ? level.slice('data:'.length) : undefined)
/** A large container's main components — the few that handle the most: `main:<container id>`. */
export const mainLevel = (id: string) => `main:${id}`
/** The context and container views draw the external systems the most containers reach; `<level>:all` is every one. */
export const overview = (level?: string) => level === 'context' || level === 'container'
export const mainOf = (level?: string) => (level?.startsWith('main:') ? level.slice('main:'.length) : undefined)
/** The container a level is inside of, whichever lens. */
export const insideOf = (level?: string) => componentOf(level) ?? dataOf(level) ?? structureOf(level) ?? mainOf(level)

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
  groups?: { name: string; label?: string; internal?: boolean; system?: boolean; stage?: boolean; rect: Rect; label_at: Point }[] | null
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
  /** One small application: the containers level would only repeat the context. */
  tiny?: boolean
  /** The system has a data-flow view. */
  dataflow?: boolean
  /** Containers with more components than a diagram shows: they open on their main ones. */
  main?: Opening[]
  /** Containers whose components are features, and so have a by-folder view too; counts their folders. */
  structure?: Opening[]
  /** How many elements the container view holds: what the system box opens onto. */
  containers: number
  /** Calls the code makes to an address computed at run time, by the container and component they leave from. */
  unresolved?: Unresolved[]
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
  commands?: number
  jobs?: number
  tables?: number
  tables_in?: string[]
  dependencies?: number
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
  /** Schema files read beside the code, whether or not the code was. */
  schemas?: string[]
  /** Configuration files read for the addresses they hold: application.yml, appsettings.json. */
  settings?: string[]
  imports: Partial<Record<'path' | 'alias' | 'module' | 'name' | 'package' | 'unresolved', number>>
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
    added_entries?: Entry[] | null
    removed_entries?: Entry[] | null
    /** The same element under another identity: renamed, or carried across the system boundary. */
    moved?: { class: 'renamed' | 're-bounded'; from: string; to: string; was: string; is: string; why: string }[] | null
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
  /** What came back, as received; absent from runs logged before it was kept. */
  response?: string
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

// ——— reading ———
// Every value comes through the place's Source (source.ts): archdoc serve on this machine, static
// files on a published site, the platform's API on the website. The paths are the same in all three.

/** Fetches path whenever it changes; null fetches nothing. Stale answers are never shown. */
export function useApi<T>(path: string | null, as: 'json' | 'text' = 'json') {
  const source = useSource()
  const { epoch } = useConnection()
  // A path that differs only by a trailing `#n` is fetched again: that is how a screen reloads.
  const [state, setState] = useState<{ path?: string; data?: T; error?: string }>({})
  // A read that failed is asked again when the place answers again, or on Try again.
  const retry = state.path === path && state.error !== undefined ? epoch : 0
  useEffect(() => {
    if (!path) return
    let live = true
    source
      .get(path)
      .then((r) => (as === 'json' ? r.json() : r.text()))
      .then(
        (data) => {
          reached()
          if (live) setState({ path, data: data as T })
        },
        (e: Error) => {
          if (unanswered(e)) unreachable()
          if (live) setState({ path, error: unanswered(e) ? notAnswering[source.capabilities.place] : e.message })
        },
      )
    return () => {
      live = false
    }
  }, [path, as, source, retry])
  const current = state.path === path
  return {
    data: current ? state.data : undefined,
    error: current ? state.error : undefined,
    loading: !!path && !current,
  }
}

/**
 * Where citations open, resolved by the place's Source: on this machine, the repository's own
 * files in the editor; on a published site, the repository host at the commit when a remote is
 * known, and plain text otherwise — never a guessed link (surface-spec S-3).
 */
export const RootContext = createContext<Place>({ root: '' })

export function useEditorLink() {
  const place = useContext(RootContext)
  const source = useSource()
  return (file: string, line?: number) => source.link(place, file, line)
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

/** A system: several repositories, and the links between them their manifests prove (§4.6). */
export interface SystemResponse {
  name: string
  repos: SystemRepo[]
  links: SystemLink[]
}

export interface SystemRepo {
  name: string
  /** Where it is on this machine: its citations open there. */
  root: string
  version: number
  commit: string
  created_at: string
  applications: number
  /** What the repository says it is for — its root manifest's description, at its line. */
  description?: string
  description_provenance?: Provenance
}

/** One repository's package requiring another's: the requirement in one, the name in the other. */
export interface SystemLink {
  from: string
  to: string
  kind: 'code'
  package: string
  version?: string
  uses: Provenance
  declares: Provenance
  /** Which of the other's code it uses: a commit or a tag, and how far the other has moved since; or the checkout beside it. */
  pin?: { kind: 'commit' | 'tag' | 'checkout'; ref: string; ahead: number }
}

/** Home: the places this machine remembers, the most recently used first (platform-surface §5.3). */
export interface HomeResponse {
  places: HomePlace[]
}

export interface HomePlace {
  id: string
  path: string
  name: string
  kind: 'repository' | 'system' | 'missing'
  /** When it was last opened in the app here; absent when it never was. */
  opened?: string
  touched: string
  /** What is wrong with a place that is missing. */
  problem?: string
  repos?: HomeRepo[]
}

export interface HomeRepo {
  name: string
  root: string
  version: number
  commit: string
  created_at: string
  /** Versions recorded since the place was last opened here; -1 when it never was. */
  since: number
  unresolved: number
  /** Its containers' kinds, in model order: what its card draws. */
  kinds?: string[]
}
