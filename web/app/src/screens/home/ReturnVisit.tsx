import type { DiffResponse, ModelResponse, PlanResponse, Version } from '../../api'
import { useApi } from '../../api'
import { useCapabilities } from '../../source'
import type { Route } from '../../route'
import { tell } from '../../story'
import { Cite } from '../../ui/Cite'
import { planLine, sinceSentence } from './since'

const day = (iso?: string) => (iso ? new Date(iso).toLocaleDateString(undefined, { day: 'numeric', month: 'short' }) : '')
/** Home shows the first few; Changes, every one (rule 15). */
const FIRST = 5

/**
 * Home on a return visit (platform-surface §17.3; app-design-screens §3): what is new since this
 * reader last looked — the versions since, told from the engine's comparison of the version last
 * seen with the latest, each change with its line and where it landed — and the plan's verdict in
 * one line. Nothing when nothing is new.
 */
export function ReturnVisit(props: { seen: number; versions: Version[]; go: (r: Partial<Route>) => void }) {
  const can = useCapabilities()
  const sorted = [...props.versions].sort((a, b) => b.id - a.id)
  const latest = sorted[0]
  const was = sorted.find((v) => v.id === props.seen)
  const newer = sorted.filter((v) => v.id > props.seen).length
  // The version last seen may be gone from this place's history; then there is nothing to compare with.
  const ready = !!latest && !!was && newer > 0
  const diff = useApi<DiffResponse>(ready ? `/api/diff?from=${props.seen}&to=${latest.id}` : null)
  const before = useApi<ModelResponse>(ready ? `/api/model?version=${props.seen}` : null)
  const after = useApi<ModelResponse>(ready ? `/api/model?version=${latest.id}` : null)
  const plan = useApi<PlanResponse>(ready && can.ask ? '/api/plan' : null)

  if (!ready || !diff.data || !before.data || !after.data) return null
  const { groups, clauses, structural } = tell(diff.data, before.data, after.data)
  const rows = groups.filter((g) => !g.title.startsWith('Wording')).flatMap((g) => g.items.map((it) => ({ ...it, kind: g.title })))
  const check = plan.data?.check

  return (
    <section className="return-visit" aria-label="Since you last looked">
      <div className="return-since">Since you last looked · {day(was.created_at)}</div>
      <p className="return-sentence">{sinceSentence(newer, structural, clauses)}</p>
      {rows.length > 0 && (
        <div className="return-rows">
          {rows.slice(0, FIRST).map((r) => (
            <div key={r.key} className="return-row">
              <span className="return-kind">{r.kind}</span>
              <span className="return-text">{r.text}</span>
              {r.cite?.file && <Cite p={r.cite} compact />}
              {r.on?.length ? (
                <button className="link-btn" onClick={() => props.go({ screen: 'change', focus: r.key, from: props.seen })}>
                  See where it landed →
                </button>
              ) : null}
            </div>
          ))}
        </div>
      )}
      <button className="link-btn" onClick={() => props.go({ screen: 'changes' })}>
        {rows.length > FIRST ? `All ${rows.length} changes, version by version, in Changes →` : 'Each version, in Changes →'}
      </button>
      {check && (
        <div className="return-plan">
          <span>{planLine(check)}</span>
          <button className="link-btn" onClick={() => props.go({ screen: 'plan' })}>
            See the check →
          </button>
        </div>
      )}
    </section>
  )
}
