// Where the app's data comes from, and what the place it runs in can do (platform-plan step 2).
//
// One app runs in three places: on this machine through `archdoc serve`, as a published site of
// static files, and on the archdoc website. The app never asks which; it asks its Source. Each place
// gives one — an adapter — and the app reads every value through it, by the paths of archdoc
// serve's API ('/api/model?v=3'), which every adapter answers in its own way.

import { createContext, useContext } from 'react'

/** What a place can do. A feature appears only where its capability is true (platform-surface §4.5). */
export interface Capabilities {
  /** Which place this is: the app on this machine, a published site, or the archdoc website. */
  place: 'local' | 'published' | 'hosted'
  /** There are versions to pick from. A published site shows one, built from the committed record. */
  history: boolean
  /** Things can be saved from the app: views and arrangement. */
  edit: boolean
  /** Each network run's requests, exactly as they were sent — which stay on the machine that sent them. */
  runDetails: boolean
  /** What archdoc plans and has not built is shown, as placeholders. */
  plans: boolean
  /** Where a citation opens: in an editor on this machine, or on the repository's host. */
  links: 'editor' | 'host'
}

/** Where a citation is resolved against: the repository on disk, or its remote at a commit. */
export interface Place {
  root: string
  remote?: string
  commit?: string
}

export interface Source {
  capabilities: Capabilities
  /** Reads an API path. Rejects with the server's message when the answer is not OK. */
  get(path: string): Promise<Response>
  /** Runs or writes something. Rejects in a place that cannot — check capabilities first. */
  act<T>(path: string, body?: unknown): Promise<T>
  /** The address a browser loads an API path from — for links and images in a page. */
  url(path: string): string
  /** Where a file and line open, or nothing where no honest link exists (surface-spec S-3). */
  link(place: Place, file: string, line?: number): string | undefined
}

const SourceContext = createContext<Source | null>(null)

export const SourceProvider = SourceContext.Provider

/** The Source of the place the app runs in. */
export function useSource(): Source {
  const s = useContext(SourceContext)
  if (!s) throw new Error('the app was started without a Source — its host must provide one')
  return s
}

/** What the place the app runs in can do. */
export const useCapabilities = () => useSource().capabilities
