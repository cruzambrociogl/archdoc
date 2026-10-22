import { useEffect, useState } from 'react'
import type { HostedRun, Runs } from '../source'
import { WebsiteBar, engineHome } from '../shell/WebsiteBar'

type Phase = NonNullable<HostedRun['phase']>

// What a read is doing, then what it did: each phase as the line that says so. The commit, once known.
const at = (c: string) => (c ? ` at ${c}` : '')
const steps: { phase: Phase; now: (c: string) => string; done: (c: string) => string }[] = [
  { phase: 'resolving', now: () => 'Finding the latest commit', done: (c) => (c ? `Found the latest commit, ${c}` : 'Found the latest commit') },
  { phase: 'cloning', now: (c) => `Fetching the code${at(c)}`, done: (c) => `Fetched the code${at(c)}` },
  { phase: 'generating', now: () => 'Reading the code and drawing the architecture', done: () => 'Read the code and drew the architecture' },
  { phase: 'storing', now: () => 'Publishing the documentation', done: () => 'Published the documentation' },
]

/**
 * Generating (platform-surface §5.2): is it working, and what is it finding? While a read runs, what
 * it has done and what it is doing; when it fails, which of the designed failures it is — each a
 * screen, not an error — and what to do next. What it is finding, part by part, waits for the engine
 * to report its progress (6.17); until then the phases are what can be said.
 */
export function Generating(props: { run?: HostedRun; error?: Error; runs: Runs; onRun: (id: number) => void; onStart: () => void }) {
  const { run: r } = props
  const working = !!r && (r.status === 'queued' || r.status === 'running')
  const now = useNow(working)
  const [again, setAgain] = useState<{ asking: boolean; error?: string }>({ asking: false })

  const repo = r ? `${r.owner}/${r.name}` : ''
  const c = r?.commit?.slice(0, 7) ?? ''
  const elapsed = r ? duration(now - Date.parse(r.started_at ?? r.queued_at)) : ''
  const doing = (p: Phase | null) => steps.find((s) => s.phase === p)?.now(c).toLowerCase()

  let q = 'What happened?'
  let sentence = ''
  let lines: string[] = []
  let act: { label: string; onClick?: () => void; href?: string; sub?: string } | undefined
  const tryAgain = (sub?: string) => ({
    label: again.asking ? 'Asking…' : 'Try again',
    sub,
    onClick: () => {
      setAgain({ asking: true })
      props.runs.request(r!.repository).then(props.onRun, (e: Error) => setAgain({ asking: false, error: e.message }))
    },
  })
  const another = { label: 'Show another repository', onClick: props.onStart }
  const machine = 'On your own machine there is no limit.'

  if (props.error || !r) {
    sentence = props.error?.message ?? ''
    act = another
  } else if (r.status === 'queued') {
    q = 'Is it working, and what is it finding?'
    sentence = `Waiting to start reading ${repo}.`
    lines = [`In the queue · ${elapsed}`]
  } else if (r.status === 'running') {
    q = 'Is it working, and what is it finding?'
    const i = Math.max(0, steps.findIndex((s) => s.phase === r.phase))
    sentence = `Reading ${repo}: ${steps[i].now(c).toLowerCase()}.`
    lines = [...steps.slice(0, i).map((s) => s.done(c)), `${steps[i].now(c)} · ${elapsed}`]
  } else if (r.failure === 'not_found') {
    q = 'Can archdoc read this?'
    sentence = `GitHub shows no public repository at ${repo}. It may be private, or not exist — private repositories need an account, which the website doesn’t have yet.`
    act = another
  } else if (r.failure === 'too_big') {
    q = 'Can archdoc read this?'
    sentence = `${repo} is too big for the website’s reader. ${machine}`
    act = { label: 'Run it on your machine', href: engineHome, sub: engineHome.replace('https://', '') }
  } else if (r.failure === 'timed_out') {
    sentence = `The read stopped at the time limit${r.phase ? ` while ${doing(r.phase)}` : ''}. Nothing was kept.`
    act = tryAgain(`or run it on your machine, where there is no limit`)
  } else if (r.failure === 'stopped') {
    sentence = `The read was interrupted partway: the reader stopped, and nothing was wrong with ${repo}. Nothing was kept.`
    act = tryAgain()
  } else {
    sentence = `The read of ${repo} failed. Nothing was kept.`
    act = tryAgain('if it keeps failing, the repository may have moved')
  }
  const failed = r?.status === 'failed'

  return (
    <div className="shell">
      <WebsiteBar onStart={props.onStart}>
        {r && (
          <nav className="crumbs" aria-label="Where you are">
            <span className="crumb current mono" aria-current="page">
              {repo}
            </span>
          </nav>
        )}
        {working && (
          <span className="status-pill gen-pill" role="status">
            <span className="gen-dot" aria-hidden="true" /> Updating…
          </span>
        )}
      </WebsiteBar>
      <main className="shell-main page">
        <div className="reading generating" aria-live="polite">
          <div className="page-q">{q}</div>
          <p className="overview-answer">{sentence}</p>
          {lines.length > 0 && (
            <ol className="gen-lines">
              {lines.map((l, i) => (
                <li key={i} className={working && i === lines.length - 1 ? 'now' : ''}>
                  {l}
                </li>
              ))}
            </ol>
          )}
          {failed && r.error && <p className="gen-error mono">{r.error}</p>}
          <div className="gen-foot">
            {act &&
              (act.href ? (
                <a className="btn btn-ink" href={act.href}>
                  {act.label}
                </a>
              ) : (
                <button className="btn btn-ink" onClick={act.onClick} disabled={again.asking}>
                  {act.label}
                </button>
              ))}
            {act?.sub && <span className="muted small">{act.sub}</span>}
            {working && <span className="muted small">Leave this open, or come back to this link; the result will be here.</span>}
          </div>
          {again.error && <p className="gen-error">{again.error}</p>}
        </div>
      </main>
    </div>
  )
}

/** The time, each second while something is under way. */
function useNow(ticking: boolean) {
  const [now, setNow] = useState(Date.now())
  useEffect(() => {
    if (!ticking) return
    const t = setInterval(() => setNow(Date.now()), 1000)
    return () => clearInterval(t)
  }, [ticking])
  return now
}

function duration(ms: number) {
  const s = Math.max(0, Math.round(ms / 1000))
  return s < 60 ? `${s} s so far` : `${Math.floor(s / 60)} min ${s % 60} s so far`
}
