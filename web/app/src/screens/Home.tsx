import { useState } from 'react'
import type { HomePlace, HomeRepo, HomeResponse } from '../api'
import { number } from '../sentence'
import { WebsiteBar } from '../shell/WebsiteBar'
import { kindStyle } from '../ui/kinds'
import { Failure, Loading, TruthMark } from '../ui/marks'

const day = (iso: string) => new Date(iso).toLocaleDateString(undefined, { day: 'numeric', month: 'short' })
const and = (items: string[]) => (items.length <= 1 ? items.join('') : `${items.slice(0, -1).join(', ')} and ${items[items.length - 1]}`)
const times = (n: number) => (n === 1 ? 'once' : n === 2 ? 'twice' : `${number(n)} times`)
const changed = (p: HomePlace) => (p.repos ?? []).some((r) => r.since > 0)

/**
 * Home, on this machine (platform-surface §5.3): did anything change since I last looked? One
 * sentence across every place this machine remembers, then a card for each — a repository, or a
 * system of them — with a status line per repository: what changed since it was last opened here,
 * what couldn't be resolved, and when it last changed. A card opens its place.
 */
export function Home(props: { data?: HomeResponse; error?: Error; missing?: boolean; onOpen: (p: HomePlace) => void; onHome: () => void }) {
  const [adding, setAdding] = useState(false)
  const places = props.data?.places ?? []
  return (
    <div className="shell">
      <WebsiteBar onStart={props.onHome}>
        <nav className="crumbs" aria-label="Where you are">
          <span className="crumb current" aria-current="page">
            Home
          </span>
        </nav>
      </WebsiteBar>
      <main className="shell-main page">
        <div className="reading wide home">
          {props.error ? (
            <Failure error={props.error.message} whole />
          ) : !props.data ? (
            <Loading q="Did anything change since I last looked?" what="the places this machine remembers" />
          ) : (
            <>
              <div className="home-head">
                <div>
                  <div className="page-q">Did anything change since I last looked?</div>
                  {props.missing && <p className="gone-more">That address names no place this machine remembers. These are the ones it does:</p>}
                  <p className="overview-answer">{places.length ? homeSentence(places) : 'No folders yet. archdoc reads one on this machine — free, and nothing is sent.'}</p>
                </div>
                <button className="btn btn-outline" aria-expanded={adding} onClick={() => setAdding((a) => !a)}>
                  Open a folder
                </button>
              </div>
              {(adding || !places.length) && <OpenAFolder />}
              <div className="home-cards">
                {places.map((p) => (
                  <Card key={p.id} p={p} onOpen={() => props.onOpen(p)} />
                ))}
              </div>
            </>
          )}
        </div>
      </main>
    </div>
  )
}

/** The sentence across every place: what changed since it was last opened, what never was, what is gone. */
function homeSentence(places: HomePlace[]): string {
  const there = places.filter((p) => p.kind !== 'missing')
  const moved = there.filter(changed).map((p) => p.name)
  const never = there.filter((p) => !p.opened).map((p) => p.name)
  const gone = places.filter((p) => p.kind === 'missing').map((p) => p.name)
  const out: string[] = []
  if (moved.length) out.push(`${and(moved)} changed the architecture since you last opened ${moved.length === 1 ? 'it' : 'them'}.`)
  else if (there.length > never.length) out.push(`Nothing changed the architecture since you last opened ${there.length - never.length === 1 ? 'it' : 'each one'}.`)
  if (never.length) out.push(`${and(never)} ${never.length === 1 ? 'hasn’t' : 'haven’t'} been opened here yet.`)
  if (gone.length) out.push(`${and(gone)} ${gone.length === 1 ? 'isn’t where it was' : 'aren’t where they were'}.`)
  return out.join(' ')
}

/** Reading a folder happens in a terminal, on this machine; the folder then appears here. */
function OpenAFolder() {
  const [copied, setCopied] = useState(false)
  const command = 'archdoc generate path/to/folder'
  return (
    <div className="home-open">
      <p>
        In a terminal, <span className="mono">{command}</span> reads a repository on this machine — free, offline, and nothing is
        sent — and it appears here when you open Home again. For several repositories in one folder, generate each, then{' '}
        <span className="mono">archdoc serve</span> the folder: it appears as a system.
      </p>
      <button
        className="btn btn-ink"
        onClick={() =>
          navigator.clipboard?.writeText(command).then(
            () => setCopied(true),
            () => setCopied(false),
          )
        }
      >
        {copied ? 'Copied' : 'Copy the command'}
      </button>
    </div>
  )
}

/** One place: what it is, where it is, and a line per repository. */
function Card({ p, onOpen }: { p: HomePlace; onOpen: () => void }) {
  const repos = p.repos ?? []
  const last = [...repos].sort((a, b) => b.created_at.localeCompare(a.created_at))[0]
  const body = (
    <>
      <div className="home-thumb" aria-hidden="true">
        {p.kind === 'system'
          ? repos.map((r) => <span key={r.root} className="home-tile home-tile-repo" />)
          : (repos[0]?.kinds ?? []).slice(0, 12).map((k, i) => <span key={i} className={`home-tile tile-${kindStyle(k).hue}`} />)}
      </div>
      <div className="home-card-body">
        <div className="home-kind">{p.kind === 'system' ? `System · ${number(repos.length)} repositories` : p.kind === 'repository' ? 'Repository' : 'Gone'}</div>
        <div className="home-name mono">{p.name}</div>
        <div className="home-path mono">{p.path}</div>
        {p.kind === 'missing' ? (
          <div className="home-line">{p.problem}</div>
        ) : (
          repos.map((r) => <Lines key={r.root} r={r} system={p.kind === 'system'} />)
        )}
        {last && (
          <div className="home-last mono">
            last changed {day(last.created_at)} · version {last.version} · {last.commit.slice(0, 7)}
          </div>
        )}
      </div>
    </>
  )
  return p.kind === 'missing' ? (
    <div className="home-card gone">{body}</div>
  ) : (
    <button className="home-card" onClick={onOpen} aria-label={`${p.name} — open it`}>
      {body}
    </button>
  )
}

/** A repository's status lines (§5.3): what changed since it was last opened here, and what couldn't be resolved. */
function Lines({ r, system }: { r: HomeRepo; system: boolean }) {
  const who = system ? `${r.name}: ` : ''
  return (
    <>
      <div className={`home-line ${r.since > 0 ? 'is-changed' : ''}`}>
        <span className="home-glyph" aria-hidden="true">
          {r.since > 0 ? 'Δ' : ''}
        </span>
        {who}
        {r.since > 0
          ? `the architecture changed ${times(r.since)} since you last opened it`
          : r.since === 0
            ? 'unchanged since you last opened it'
            : 'not opened here yet'}
      </div>
      {r.unresolved > 0 && (
        <div className="home-line">
          <TruthMark state="unresolved" /> {who}
          {r.unresolved} {r.unresolved === 1 ? 'thing' : 'things'} couldn’t be resolved
        </div>
      )}
    </>
  )
}
