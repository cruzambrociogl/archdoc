import type { ComponentType } from 'react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import type { HostedRun, Runs } from '../source'
import { SourceProvider } from '../source'
import { Start } from '../screens/Start'
import { Generating } from '../screens/Generating'
import { Loading } from '../ui/marks'
import type { Notice } from './Notice'
import { NoticeProvider } from './Notice'

/** Where on the website: Start (`/`, `/new?repo=…`), or one read (`/runs/{id}`). After `#`, the repository's own pages. */
type Where = { page: 'start'; repo?: string } | { page: 'run'; id: number }

function where(): Where {
  const run = location.pathname.match(/^\/runs\/(\d+)\/?$/)
  if (run) return { page: 'run', id: Number(run[1]) }
  const repo = location.pathname === '/new' ? (new URLSearchParams(location.search).get('repo') ?? undefined) : undefined
  return { page: 'start', repo }
}

const finished = (r: HostedRun) => r.status === 'done' || r.status === 'unchanged' || r.status === 'failed'

/**
 * The website's pages before a repository's (platform-surface §5.1–5.2): Start, then a read until it
 * finishes — and then that repository's documentation, every page, through the Source the finished
 * run opens. The path says which; the repository's pages keep their own addresses after `#`, as in
 * every other place.
 */
export function Website({ runs, Repository }: { runs: Runs; Repository: ComponentType }) {
  const [at, setAt] = useState(where)
  useEffect(() => {
    const back = () => setAt(where())
    window.addEventListener('popstate', back)
    return () => window.removeEventListener('popstate', back)
  }, [])
  const open = useCallback((path: string, replace = false) => {
    history[replace ? 'replaceState' : 'pushState'](null, '', path)
    setAt(where())
  }, [])
  const start = useCallback(() => open('/'), [open])
  const toRun = useCallback((id: number, replace = false) => open(`/runs/${id}`, replace), [open])

  if (at.page === 'start') return <Start key={at.repo} repo={at.repo} runs={runs} onRun={toRun} onStart={start} />
  return <Run key={at.id} id={at.id} runs={runs} onRun={toRun} onStart={start} Repository={Repository} />
}

/** One read: followed each second until it finishes, then its documentation or its failure. */
function Run(props: { id: number; runs: Runs; onRun: (id: number) => void; onStart: () => void; Repository: ComponentType }) {
  const { id, runs, Repository } = props
  const [run, setRun] = useState<HostedRun>()
  const [error, setError] = useState<Error>()
  useEffect(() => {
    let stopped = false
    let timer: ReturnType<typeof setTimeout> | undefined
    const follow = () =>
      runs.get(id).then(
        (r) => {
          if (stopped) return
          setRun(r)
          if (!finished(r)) timer = setTimeout(follow, 1000)
        },
        (e: Error) => !stopped && setError(e),
      )
    void follow()
    return () => {
      stopped = true
      clearTimeout(timer)
    }
  }, [id, runs])

  const documented = run && (run.status === 'done' || run.status === 'unchanged') ? run : undefined
  const source = useMemo(() => documented && runs.open(documented), [documented, runs])
  const notices = useMemo(() => (documented ? unchanged(documented) : []), [documented])

  if (source) {
    return (
      <SourceProvider value={source}>
        <NoticeProvider value={notices}>
          <Repository />
        </NoticeProvider>
      </SourceProvider>
    )
  }
  if (!run && !error) {
    return (
      <main className="boot">
        <Loading what="the run" />
      </main>
    )
  }
  return <Generating run={run} error={error} runs={runs} onRun={props.onRun} onStart={props.onStart} />
}

const day = (iso: string) => new Date(iso).toLocaleDateString(undefined, { day: 'numeric', month: 'short' })

/** A run that found the architecture unchanged shows the version before it, and says so. */
function unchanged(r: HostedRun): Notice[] {
  if (r.status !== 'unchanged' || !r.version_commit) return []
  return [
    {
      id: `unchanged-${r.id}`,
      glyph: 'info',
      what: (
        <>
          The architecture hasn’t changed since these docs were made, from commit <span className="mono">{r.version_commit.slice(0, 7)}</span>
          {r.version_finished_at ? ` on ${day(r.version_finished_at)}` : ''}.
        </>
      ),
      still: (
        <>
          archdoc read {r.owner}/{r.name} again{r.commit ? <> at <span className="mono">{r.commit.slice(0, 7)}</span></> : ''}
          {r.finished_at ? ` on ${day(r.finished_at)}` : ''} and found the same parts and connections.
        </>
      ),
    },
  ]
}
