import type { ReactNode } from 'react'
import { useMemo } from 'react'
import type { DiffResponse, ModelResponse, SystemRepo, SystemResponse, Version } from '../api'
import { useApi } from '../api'
import { number, pinWords } from '../sentence'
import type { Crumb } from '../shell/parent'
import type { Systems } from '../source'
import { SourceProvider } from '../source'
import { join, tell } from '../story'
import { Failure, Loading } from '../ui/marks'
import { SystemFrame } from './SystemMap'

const day = (iso: string) => new Date(iso).toLocaleDateString(undefined, { day: 'numeric', month: 'short' })

/**
 * What changed across a system (platform-surface §4.6): its repositories' change stories together,
 * the most recent first — each one's latest version in the sentence its own What changed writes,
 * and, where another repository of the system requires it, who — the first of "a change in one
 * repository marked on the repositories that depend on it".
 */
export function SystemChanges(props: {
  sys?: SystemResponse
  error?: Error
  above?: Crumb[]
  system: Systems
  onMap: () => void
  onChanges: () => void
  onOpen: (repo: string, page?: string) => void
}) {
  const { sys } = props
  const repos = sys ? [...sys.repos].sort((a, b) => b.created_at.localeCompare(a.created_at) || a.name.localeCompare(b.name)) : []
  // One Source per repository, kept: a new one each render would read everything again.
  const sources = useMemo(() => new Map((sys?.repos ?? []).map((r) => [r.name, props.system.open(r.name)])), [sys, props.system])
  return (
    <SystemFrame sys={sys} above={props.above} on="changes" onMap={props.onMap} onChanges={props.onChanges}>
      {props.error ? (
        <Failure error={props.error.message} whole />
      ) : !sys ? (
        <Loading q="What changed across them?" what="the repositories" />
      ) : (
        <>
          <div className="page-q">What changed across them?</div>
          <p className="overview-answer">{acrossSentence(repos)}</p>
          {repos.map((r) => (
            <SourceProvider key={r.name} value={sources.get(r.name)!}>
              <RepoChange r={r} sys={sys} onOpen={props.onOpen} />
            </SourceProvider>
          ))}
        </>
      )}
    </SystemFrame>
  )
}

/** Which changed last, and which have nothing to compare yet — from each repository's latest version. */
function acrossSentence(repos: SystemRepo[]): string {
  const changed = repos.filter((r) => r.version > 1)
  const first = repos.filter((r) => r.version === 1).map((r) => r.name)
  const out: string[] = []
  if (changed.length) {
    const [last, ...rest] = changed
    out.push(`${last.name} changed most recently: its version ${last.version}, on ${day(last.created_at)}.`)
    if (rest.length) out.push(`${join(rest.map((r) => `${r.name} last changed on ${day(r.created_at)}`))}.`)
  }
  if (first.length) out.push(`${join(first)} ${first.length === 1 ? 'has' : 'have'} ${number(1)} version so far: nothing to compare with yet.`)
  return out.join(' ')
}

/** One repository's latest version, told in a sentence, with who in the system requires it. */
function RepoChange({ r, sys, onOpen }: { r: SystemRepo; sys: SystemResponse; onOpen: (repo: string, page?: string) => void }) {
  const versions = useApi<Version[]>('/api/versions')
  const ids = (versions.data ?? []).map((v) => v.id).sort((a, b) => b - a)
  const [to, base] = ids
  const diff = useApi<DiffResponse>(base ? `/api/diff?from=${base}&to=${to}` : null)
  const before = useApi<ModelResponse>(base ? `/api/model?version=${base}` : null)
  const after = useApi<ModelResponse>(base ? `/api/model?version=${to}` : null)
  const dependents = sys.links.filter((l) => l.to === r.name)
  const by = [...new Set(dependents.map((l) => l.from))]

  let story: ReactNode
  if (versions.error || diff.error) story = <Failure error={(versions.error ?? diff.error)!} />
  else if (!versions.data) story = <Loading what={`${r.name}'s versions`} />
  else if (!base) story = <p className="system-change-line">Its first version: there is nothing before it to compare with.</p>
  else if (!diff.data || !before.data || !after.data) story = <Loading what={`what changed in ${r.name}`} />
  else {
    const told = tell(diff.data, before.data, after.data)
    const groups = told.groups.filter((g) => g.items.length && !g.title.startsWith('Wording'))
    story = (
      <>
        <p className="system-change-line">{told.structural ? `Its latest version ${told.clauses.join('; ')}.` : 'Its latest version changed code, but not the architecture.'}</p>
        {groups.length > 0 && (
          <div className="system-change-groups">
            {groups.map((g) => (
              <span key={g.title} className="system-change-group">
                <span className="meaning-tag">{g.tag}</span> {g.title} · {g.items.length}
              </span>
            ))}
          </div>
        )}
      </>
    )
  }

  return (
    <section className="system-change">
      <h2 className="system-change-head">
        <span className="mono">{r.name}</span>
        <span className="muted small mono">
          version {r.version} · {r.commit.slice(0, 7)} · {day(r.created_at)}
        </span>
      </h2>
      {story}
      {dependents.length > 0 && (
        <p className="system-change-deps muted">
          {join(by)} {by.length === 1 ? 'requires' : 'require'} it:{' '}
          {dependents.map((l) => `${l.package}${pinWords(l)}`).join('; ')}.
        </p>
      )}
      <button className="link-btn" onClick={() => onOpen(r.name, 'changes')}>
        {r.name}’s What changed →
      </button>
    </section>
  )
}
