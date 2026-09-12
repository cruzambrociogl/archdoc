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
}

export interface Model {
  name: string
  source: string
  nodes: Node[] | null
  edges: Edge[] | null
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
  rules: {
    line: number
    set: Record<string, string> | null
    exclude: boolean
    remove: boolean
    match: { name: string; image: string; kind: string }
    edge?: { from: string; to: string }
  }[]
  operations: Op[] | null
  findings: { rule: number; severity: string; element: string; message: string }[]
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

async function request(path: string): Promise<Response> {
  const r = await fetch(path)
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

/** The repository's absolute path, so every citation can open in the editor. */
export const RootContext = createContext('')

export function editorLink(root: string, file: string, line?: number): string {
  const path = file.startsWith('/') ? file : `${root}/${file}`
  return `vscode://file${path}${line ? `:${line}` : ''}`
}

export function useEditorLink() {
  const root = useContext(RootContext)
  return (file: string, line?: number) => editorLink(root, file, line)
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
