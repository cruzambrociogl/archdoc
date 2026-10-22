import { useEffect, useMemo, useState } from 'react'
import type { DiffResponse, ModelResponse, Session, Version } from '../api'
import type { Item } from '../story'
import { join, tell } from '../story'
import { useApi } from '../api'
import { useCapabilities } from '../source'
import type { Route } from '../route'
import { Cite } from '../ui/Cite'
import { Failure, Loading } from '../ui/marks'
import { Changes } from './Changes'

/**
 * What changed (platform-surface §6.2): what just changed in the architecture? A strip of the
 * versions — which is also the picker between them — then one version told as a story, its changes
 * grouped by what they mean, each with its citation in view. Without AI the story is written by a
 * template from the engine's diff, so it is proven. The detailed comparison of any two versions is
 * still here, under Advanced, intact.
 */
export function WhatChanged(props: { versions: Version[]; route: Route; go: (r: Partial<Route>, o?: { keep?: boolean }) => void; root: string; session?: Session }) {
  const can = useCapabilities()
  const [advanced, setAdvanced] = useState(props.route.from !== undefined)
  const sorted = useMemo(() => [...props.versions].sort((a, b) => b.id - a.id), [props.versions])
  const to = props.route.v ?? sorted[0]?.id
  const base = sorted.find((v) => to !== undefined && v.id < to)?.id
  const ready = !!(to && base)
  const diff = useApi<DiffResponse>(ready ? `/api/diff?from=${base}&to=${to}` : null)
  const before = useApi<ModelResponse>(ready ? `/api/model?version=${base}` : null)
  const after = useApi<ModelResponse>(to ? `/api/model?version=${to}` : null)
  const seen = useSeen(props.root, sorted[0]?.id)

  const newer = seen === undefined ? 0 : sorted.filter((v) => v.id > seen).length
  const since =
    can.place !== 'local'
      ? 'since the version before it'
      : seen === undefined
        ? ''
        : newer
          ? `${newer} ${newer === 1 ? 'version changed' : 'versions changed'} the architecture since you last opened it here`
          : 'Nothing has changed since you last opened it here'

  return (
    <div className="reading what-changed">
      <div className="page-q">What just changed?</div>

      {sorted.length > 1 && (
        <div className="version-strip" role="group" aria-label="Versions">
          {since && <div className="version-strip-since">{since}</div>}
          <div className="version-strip-row">
            {sorted.slice(0, 8).map((v) => (
              <button
                key={v.id}
                className={`version-chip ${v.id === to ? 'on' : ''} ${seen !== undefined && v.id > seen ? 'new' : ''}`}
                aria-pressed={v.id === to}
                onClick={() => props.go({ screen: 'changes', v: v.id === sorted[0].id ? undefined : v.id })}
              >
                <span className="version-chip-day">{day(v.created_at)}</span>
                <span className="mono version-chip-commit">{v.commit ? v.commit.slice(0, 7) : `v${v.id}`}</span>
              </button>
            ))}
          </div>
        </div>
      )}

      {!ready ? (
        <p className="story-answer">
          This is the first version archdoc recorded{to ? ` (${label(sorted.find((v) => v.id === to))})` : ''}: there is nothing before it to compare with. The next version that changes the architecture
          will be told here.
        </p>
      ) : diff.error ? (
        <Failure error={diff.error} />
      ) : !diff.data || !after.data || !before.data ? (
        <Loading what="what changed" />
      ) : (
        <Story d={diff.data} before={before.data} after={after.data} label={label(sorted.find((v) => v.id === to))} go={props.go} base={base!} to={to!} />
      )}

      {sorted.length > 1 && (
        <div className="advanced">
          <button className="evidence-toggle" aria-expanded={advanced} onClick={() => setAdvanced((a) => !a)}>
            <span className="caret">{advanced ? '▾' : '▸'}</span> Advanced <span className="muted">every version, and comparing any two</span>
          </button>
          {advanced && (
            <div className="advanced-body">
              <Changes versions={props.versions} route={props.route} go={props.go} />
            </div>
          )}
        </div>
      )}
    </div>
  )
}

/** One version, told: a sentence, then its changes by what they mean (§6.2). */
function Story(props: { d: DiffResponse; before: ModelResponse; after: ModelResponse; label: string; base: number; to: number; go: (r: Partial<Route>) => void }) {
  const { groups, clauses, structural } = tell(props.d, props.before, props.after)
  const full = groups.filter((g) => g.items.length)
  const empty = groups.filter((g) => !g.items.length && !g.title.startsWith('Wording'))
  return (
    <>
      <p className="story-answer">
        {structural ? `This version (${props.label}) ${clauses.join('; ')}.` : `This version (${props.label}) changed code, but not the architecture.`}
      </p>
      <div className="overview-source">
        <span>Written from the engine's comparison with the version before — proven, not interpreted.</span>
        {structural && (
          <button className="link-btn" onClick={() => props.go({ screen: 'explorer', level: 'container', v: props.to === undefined ? undefined : props.to, from: props.base })}>
            See it on the map →
          </button>
        )}
      </div>
      <div className="meanings">
        {full.map((g) => (
          <section key={g.title} className={`meaning-group ${g.title.startsWith('Wording') ? 'wording' : ''}`}>
            <h2 className="meaning-head">
              <span className="meaning-tag">{g.tag}</span>
              {g.title}
              <span className="muted"> · {g.items.length}</span>
            </h2>
            <ul>
              {g.items.slice(0, 5).map((it) => (
                <li key={it.key}>
                  <span className="meaning-text">{it.text}</span>
                  {it.cite?.file && <Cite p={it.cite} compact />}
                </li>
              ))}
            </ul>
            {g.items.length > 5 && <MoreItems items={g.items.slice(5)} />}
          </section>
        ))}
        {empty.length > 0 && <p className="meaning-nothing">{nothingLine(empty.map((g) => g.title))}</p>}
      </div>
    </>
  )
}

/** Lists stop at five (rule 15); the rest is one deliberate step away. */
function MoreItems({ items }: { items: Item[] }) {
  const [open, setOpen] = useState(false)
  if (!open)
    return (
      <button className="link-btn meaning-more" onClick={() => setOpen(true)}>
        {items.length} more →
      </button>
    )
  return (
    <ul>
      {items.map((it) => (
        <li key={it.key}>
          <span className="meaning-text">{it.text}</span>
          {it.cite?.file && <Cite p={it.cite} compact />}
        </li>
      ))}
    </ul>
  )
}

/** The empty groups, in one closing line (App Findings, round 5): "No new outside service; nothing removed." */
function nothingLine(titles: string[]): string {
  const phrase: Record<string, string> = {
    'New capability': 'no new capability',
    'New outside service': 'no new outside service',
    'Data changes': 'no data change',
    Changed: 'nothing changed',
    Removed: 'nothing removed',
  }
  const s = join(titles.map((t) => phrase[t] ?? t.toLowerCase()))
  return s.charAt(0).toUpperCase() + s.slice(1) + '.'
}

const day = (iso: string) => new Date(iso).toLocaleDateString(undefined, { day: 'numeric', month: 'short' })
const label = (v?: Version) => (v ? `version ${v.id}${v.commit ? ` · ${v.commit.slice(0, 7)}` : ''} · ${day(v.created_at)}` : '')

/**
 * Which version this browser last showed for this repository (§4.5: locally, "since you last
 * opened it on this machine"). Read once, then moved to the latest. Storage may be blocked; then
 * there is no "since", only the versions.
 */
function useSeen(root: string, latest?: number): number | undefined {
  const key = `archdoc.seen.${root}`
  const [seen] = useState<number | undefined>(() => {
    try {
      const v = localStorage.getItem(key)
      return v === null ? undefined : Number(v)
    } catch {
      return undefined
    }
  })
  useEffect(() => {
    if (latest === undefined) return
    try {
      localStorage.setItem(key, String(latest))
    } catch {
      /* blocked: nothing remembered */
    }
  }, [key, latest])
  return seen
}
