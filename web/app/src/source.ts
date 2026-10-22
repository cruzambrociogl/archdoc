// Where the app's data comes from, and what the place it runs in can do (platform-plan step 2).
//
// One app runs in three places: on this machine through `archdoc serve`, as a published site of
// static files, and on the archdoc website. The app never asks which; it asks its Source. Each place
// gives one — an adapter — and the app reads every value through it, by the paths of archdoc
// serve's API ('/api/model?v=3'), which every adapter answers in its own way.

import { createContext, useContext } from 'react'
import type { HomePlace, HomeResponse, SystemResponse, Version } from './api'

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
  /** A model can be asked, on the person's own key: Ask, and Did it build what I asked? (§6.3, §6.5). */
  ask: boolean
}

/** Where a citation is resolved against: the repository on disk, or its remote at a commit. */
export interface Place {
  root: string
  remote?: string
  commit?: string
}

export interface Source {
  capabilities: Capabilities
  /**
   * Reads an API path. Rejects with the server's message when the answer is not OK, and with fetch's
   * TypeError when nothing answers — which the app tells apart (connection.ts).
   */
  get(path: string): Promise<Response>
  /** Runs or writes something. Rejects in a place that cannot — check capabilities first. */
  act<T>(path: string, body?: unknown): Promise<T>
  /** The address a browser loads an API path from — for links and images in a page. */
  url(path: string): string
  /** Where a file and line open, or nothing where no honest link exists (surface-spec S-3). */
  link(place: Place, file: string, line?: number): string | undefined
  /**
   * Where a repository is read on request: the archdoc website. Given, the app opens on Start and
   * follows a read to its documentation (platform-surface §5.1–5.2); absent, it shows one repository.
   */
  runs?: Runs
  /**
   * Several repositories read together: on this machine, a folder of them (platform-surface §4.6).
   * Given, the app opens on the system's map, and each repository's pages under its name; absent, it
   * shows one repository.
   */
  system?: Systems
  /**
   * The places this machine remembers (platform-surface §5.3). Given, the app opens on Home, and
   * each place — a repository, or a system — under it.
   */
  home?: Homes
}

/** Home's places, each one's own Source, and remembering that one was opened. */
export interface Homes {
  get(): Promise<HomeResponse>
  open(place: HomePlace): Source
  opened(id: string): Promise<void>
}

/** A system's repositories, and each one's own Source. */
export interface Systems {
  get(): Promise<SystemResponse>
  open(repo: string): Source
}

/** Reading repositories on request — the website's runs. */
export interface Runs {
  /** Asks for a repository by its link; the run's number. Rejects with a Refusal. */
  request(link: string): Promise<number>
  /** A run as it stands. */
  get(id: number): Promise<HostedRun>
  /** A repository the website has been asked for, by its address; undefined when it never was. */
  repository(owner: string, name: string): Promise<HostedRepository | undefined>
  /** The Source of a repository's documentation: its latest version, and every one before it. */
  openRepository(repo: HostedRepository): Source
}

/** A repository as the website addresses it, /{owner}/{name}: its versions, newest first, and its latest run. */
export interface HostedRepository {
  owner: string
  name: string
  url: string
  versions: Version[]
  /** The most recent read, whatever became of it: one on its way, or one that found nothing changed. */
  latest_run?: HostedRun
}

/** A request turned down before anything ran, with why as a kind and in plain words. */
export class Refusal extends Error {
  constructor(
    message: string,
    readonly kind: 'not_github' | 'not_found' | 'unavailable',
  ) {
    super(message)
  }
}

/** One read of one repository, as the website reports it. */
export interface HostedRun {
  id: number
  /** The repository's link, its owner and name. */
  repository: string
  owner: string
  name: string
  status: 'queued' | 'running' | 'done' | 'unchanged' | 'failed'
  /** What a running read is doing. */
  phase: 'resolving' | 'cloning' | 'generating' | 'storing' | null
  /** Which of the designed failures a failed read is (§5.2): missing or private, too big, out of time, interrupted, other. */
  failure: 'not_found' | 'too_big' | 'timed_out' | 'stopped' | 'failed' | null
  error: string | null
  commit: string | null
  queued_at: string
  started_at: string | null
  finished_at: string | null
  /** The version its documentation is: its own, or, unchanged, the one before it — and that version's commit and time. */
  version: number | null
  version_commit?: string | null
  version_finished_at?: string | null
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
