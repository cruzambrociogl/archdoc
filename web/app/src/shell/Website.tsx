import type { ComponentType } from 'react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import type { HostedRepository, HostedRun, Runs } from '../source'
import { SourceProvider } from '../source'
import { Start } from '../screens/Start'
import { Generating } from '../screens/Generating'
import { Loading } from '../ui/marks'
import type { Notice } from './Notice'
import { NoticeProvider } from './Notice'
import { ParentContext } from './parent'
import { goTo, replaceWith, usePath } from './path'
import { address, atOf, lineKind, lineLabel } from './line'
import type { At } from '../source'

/**
 * Where on the website: Start (`/`, `/new?repo=…`), one read (`/runs/{id}`), or a repository
 * (`/{owner}/{name}`), at a branch or tag (`…/tree/{name}`) or a commit (`…/commit/{hash}`), as
 * GitHub's own links are made. After `#`, the repository's own pages, as in every other place.
 */
type Where = { page: 'start'; repo?: string } | { page: 'run'; id: number } | { page: 'repo'; owner: string; name: string; at?: At }

// First segments that are the website's own, never an owner's.
const reserved = new Set(['new', 'runs', 'api', 'assets'])

function where(path: string): Where {
  const run = path.match(/^\/runs\/(\d+)\/?$/)
  if (run) return { page: 'run', id: Number(run[1]) }
  const repo = path.match(/^\/([A-Za-z0-9._-]+)\/([A-Za-z0-9._-]+)(?:\/(tree\/.+?|commit\/[0-9a-f]{40}))?\/?$/)
  if (repo && !reserved.has(repo[1])) {
    const [kind, rest] = (repo[3] ?? '').split(/\/(.*)/s)
    const at = kind === 'tree' ? { tree: rest.split('/').map(decodeURIComponent).join('/') } : kind === 'commit' ? { commit: rest } : undefined
    return { page: 'repo', owner: repo[1], name: repo[2], at }
  }
  return { page: 'start', repo: path === '/new' ? (new URLSearchParams(location.search).get('repo') ?? undefined) : undefined }
}

const finished = (r: HostedRun) => r.status === 'done' || r.status === 'unchanged' || r.status === 'failed'

/**
 * The website's pages (platform-surface §5.1–5.2): Start; a read until it finishes; and a repository
 * at its own address — its latest documentation, every version before it, every page — through the
 * Source the website opens for it. A finished read goes to its repository's address.
 */
export function Website({ runs, Repository }: { runs: Runs; Repository: ComponentType }) {
  const at = where(usePath())
  const start = useCallback(() => goTo('/'), [])
  const toRun = useCallback((id: number, replace = false) => (replace ? replaceWith : goTo)(`/runs/${id}`), [])

  if (at.page === 'start') return <Start key={at.repo} repo={at.repo} runs={runs} onRun={toRun} onStart={start} />
  if (at.page === 'repo') {
    return <RepositoryPage key={address(at.owner, at.name, at.at)} owner={at.owner} name={at.name} at={at.at} runs={runs} onRun={toRun} onStart={start} Repository={Repository} />
  }
  return <Run key={at.id} id={at.id} runs={runs} onRun={toRun} onStart={start} />
}

/**
 * One read: followed each second until it finishes; then its failure, or its repository's address —
 * at this read's version when a later one has come since, so an old link still shows what it showed.
 */
function Run(props: { id: number; runs: Runs; onRun: (id: number) => void; onStart: () => void }) {
  const { id, runs } = props
  const [run, setRun] = useState<HostedRun>()
  const [error, setError] = useState<Error>()
  useEffect(() => {
    let stopped = false
    let timer: ReturnType<typeof setTimeout> | undefined
    const follow = () =>
      runs.get(id).then(
        async (r) => {
          if (stopped) return
          if (!finished(r)) {
            setRun(r)
            timer = setTimeout(follow, 1000)
            return
          }
          if (r.status === 'failed' || !r.version) return setRun(r)
          const repo = await runs.repository(r.owner, r.name, atOf(r.ref)).catch(() => undefined)
          if (stopped) return
          const latest = repo?.versions[0]?.id
          replaceWith(`${address(r.owner, r.name, atOf(r.ref))}${latest !== undefined && latest !== r.version ? `#/overview?v=${r.version}` : ''}`)
        },
        (e: Error) => !stopped && setError(e),
      )
    void follow()
    return () => {
      stopped = true
      clearTimeout(timer)
    }
  }, [id, runs])

  if (!run && !error) {
    return (
      <main className="boot">
        <Loading what="the run" />
      </main>
    )
  }
  return <Generating run={run} error={error} runs={runs} onRun={props.onRun} onStart={props.onStart} />
}

/**
 * A repository at its address: its documentation, with what its latest read says — one on its way,
 * or one that found nothing changed. One never asked for offers to be read; it is not read unasked.
 */
function RepositoryPage(props: { owner: string; name: string; at?: At; runs: Runs; onRun: (id: number) => void; onStart: () => void; Repository: ComponentType }) {
  const { owner, name, at, runs, Repository } = props
  const [repo, setRepo] = useState<HostedRepository | null>()
  const [error, setError] = useState<Error>()
  useEffect(() => {
    runs.repository(owner, name, at).then((r) => setRepo(r ?? null), (e: Error) => setError(e))
  }, [owner, name, at, runs])

  const versions = repo?.versions.length ?? 0
  const running = repo?.latest_run && !finished(repo.latest_run) ? repo.latest_run : undefined
  const source = useMemo(() => (repo && versions ? runs.openRepository(repo) : undefined), [repo, versions, runs])
  const notices = useMemo(() => (repo ? latestRead(repo, props.onRun) : []), [repo, props.onRun])
  // Which line is shown, and the others read, each at its address.
  const parent = useMemo(() => {
    const line = repo
      ? {
          label: lineLabel(repo.ref, repo.default_branch),
          kind: lineKind(repo.ref),
          others: (repo.lines ?? [])
            .filter((l) => l.ref !== (repo.ref ?? ''))
            .map((l) => ({ label: lineLabel(l.ref, repo.default_branch), kind: lineKind(l.ref), open: () => goTo(address(repo.owner, repo.name, atOf(l.ref))) })),
        }
      : undefined
    return { crumbs: [], repo: `${repo?.owner ?? owner}/${repo?.name ?? name}`, line }
  }, [repo, owner, name])

  // Nothing documented yet, but a read on its way: follow it.
  useEffect(() => {
    if (repo && !versions && running) replaceWith(`/runs/${running.id}`)
  }, [repo, versions, running])

  if (error) return <Generating error={error} runs={runs} onRun={props.onRun} onStart={props.onStart} />
  if (repo === undefined || (!versions && running)) {
    return (
      <main className="boot">
        <Loading what={`${owner}/${name}`} />
      </main>
    )
  }
  if (!source) return <Start suggest={`https://github.com${address(owner, name, at)}`} runs={runs} onRun={props.onRun} onStart={props.onStart} />
  return (
    <SourceProvider value={source}>
      <NoticeProvider value={notices}>
        <ParentContext.Provider value={parent}>
          <Repository />
        </ParentContext.Provider>
      </NoticeProvider>
    </SourceProvider>
  )
}

const day = (iso: string) => new Date(iso).toLocaleDateString(undefined, { day: 'numeric', month: 'short' })

/** What the latest read says, above the documentation: on its way, or nothing changed since. */
function latestRead(repo: HostedRepository, onRun: (id: number) => void): Notice[] {
  const r = repo.latest_run
  const latest = repo.versions[0]
  if (!r || !latest) return []
  if (!finished(r)) {
    return [
      {
        id: `reading-${r.id}`,
        glyph: 'info',
        what: `archdoc is reading ${repo.owner}/${repo.name} again.`,
        still: 'These docs stay as they are until it finishes; then the newer version is here.',
        acts: [{ label: 'Follow it', on: () => onRun(r.id), primary: true }],
      },
    ]
  }
  if (r.status !== 'unchanged' || r.version !== latest.id) return []
  return [
    {
      id: `unchanged-${r.id}`,
      glyph: 'info',
      what: (
        <>
          The architecture hasn’t changed since these docs were made, from commit <span className="mono">{latest.commit.slice(0, 7)}</span> on{' '}
          {day(latest.created_at)}.
        </>
      ),
      still: (
        <>
          archdoc read {repo.owner}/{repo.name} again{r.commit ? <> at <span className="mono">{r.commit.slice(0, 7)}</span></> : ''}
          {r.finished_at ? ` on ${day(r.finished_at)}` : ''} and found the same parts and connections.
        </>
      ),
    },
  ]
}
