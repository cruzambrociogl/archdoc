import { useState } from 'react'
import type { CoverageResponse } from '../api'
import { useApi } from '../api'
import type { Route } from '../route'
import { Eyebrow, Failure, Loading, TruthMark } from '../ui/marks'

const ruleNames: Record<string, string> = {
  'VAL-03': 'Relationships that state no protocol',
  'VAL-06': 'Elements missing a description or a technology',
}

/**
 * What archdoc could not see (surface-spec §5.11) — the page no competitor publishes. Nothing here
 * is interpretation: every line is a count of facts, a validator finding, or a stated limit of the
 * evidence. It is the same report as the committed coverage page, read from .archdoc/coverage.json.
 */
export function Coverage(props: { go: (r: Partial<Route>) => void }) {
  const c = useApi<CoverageResponse>('/api/coverage')
  const [open, setOpen] = useState<Record<string, boolean>>({})
  if (c.error) return <Failure error={c.error} />
  if (!c.data) return <Loading />
  const r = c.data
  const gapCount = r.gaps.reduce((s, g) => s + g.gaps.length, 0)
  const toElement = (element: string) => {
    const edge = element.split(' → ')
    props.go({ screen: 'explorer', level: 'container', focus: edge.length === 2 ? `${edge[0]}>${edge[1]}` : element })
  }

  return (
    <div className="reading wide">
      <Eyebrow>Coverage · what archdoc could not see</Eyebrow>
      <h1 className="page-title">Where the proof runs out</h1>
      <p className="lede">
        Everything in the other screens is proven. This page says where the proof stops: {gapCount === 0 ? 'no gaps were found' : `${gapCount} gaps`}, what
        nothing connects to, what was read and passed over, and what reading {r.source ? <span className="mono">{r.source}</span> : 'configuration'} cannot state at all.
      </p>

      <div className="stat-strip">
        {r.known.map((k) => (
          <div key={k.label} className="stat static">
            <span className="stat-n">{k.value}</span>
            <span className="stat-label">{k.label}</span>
          </div>
        ))}
      </div>

      <div className="coverage-grid">
        <section>
          <Eyebrow>Could not be resolved · {gapCount}</Eyebrow>
          {r.gaps.length === 0 && <p className="lede-quiet">Nothing. Every element and relationship the configuration states is complete.</p>}
          {r.gaps.map((g) => {
            const all = open[g.rule]
            const shown = all ? g.gaps : g.gaps.slice(0, 12)
            return (
              <div key={g.rule} className="gap-group">
                <div className="gap-head">
                  <TruthMark state="unresolved" />
                  <span className="gap-title">{ruleNames[g.rule] ?? g.rule}</span>
                  <span className="mono small">
                    {g.rule} · {g.gaps.length}
                  </span>
                </div>
                {shown.map((gap, i) => (
                  <button key={i} className="gap-row" onClick={() => toElement(gap.element)} title="Open in the explorer">
                    <span className="mono">{gap.element}</span>
                    <span className="muted">{gap.message}</span>
                  </button>
                ))}
                {g.gaps.length > shown.length && (
                  <button className="text-link gap-more" onClick={() => setOpen((o) => ({ ...o, [g.rule]: true }))}>
                    Show all {g.gaps.length}
                  </button>
                )}
              </div>
            )
          })}
          {r.unconnected.length > 0 && (
            <div className="gap-group">
              <div className="gap-head">
                <TruthMark state="unresolved" />
                <span className="gap-title">Elements nothing reaches and that reach nothing</span>
                <span className="mono small">{r.unconnected.length}</span>
              </div>
              <p className="gap-note">
                This is rarely true of a running system. It usually means the connection is made in application code or at deploy time,
                where configuration cannot see it. The element is drawn unconnected rather than joined up on a guess.
              </p>
              <div className="gap-row static">
                <span className="mono">{r.unconnected.join(', ')}</span>
              </div>
            </div>
          )}
        </section>

        <section>
          <Eyebrow>What was read</Eyebrow>
          {r.read.map((f) => (
            <div key={f.file} className="read-row">
              <div className="mono small">
                {f.file} <span className={f.used ? '' : 'muted'}>· {f.used ? 'used' : 'passed over'}</span>
              </div>
              <div className="read-why">{f.why}</div>
            </div>
          ))}
          <div className="eyebrow coverage-sub">What configuration cannot state</div>
          <p className="lede-quiet">These are not gaps in this repository. They are the limits of the evidence.</p>
          {r.limits.map((l) => (
            <div key={l.limit} className="read-row">
              <div className="small">{l.limit.replace(/\*/g, '')}</div>
              <div className="read-why">needs: {l.needed}</div>
            </div>
          ))}
        </section>
      </div>
    </div>
  )
}
