import { useState } from 'react'
import type { Claim, PlanCheck as Check, PlanResponse } from '../api'
import { useApi } from '../api'
import type { Route } from '../route'
import { number } from '../sentence'
import { useSource } from '../source'
import { Cite } from '../ui/Cite'
import { Failure, Loading, TruthMark } from '../ui/marks'
import { aiFailure, dollars } from './Ask'

const day = (iso?: string) => (iso ? new Date(iso).toLocaleDateString(undefined, { day: 'numeric', month: 'short' }) : '')

/**
 * Did it build what I asked? (platform-surface §6.3): the plan files a person named, requirement by
 * requirement — looks built, with the code it matched; looks missing, with what was looked for;
 * couldn't check, with why — and what was built that the plan didn't ask for. Matching a sentence to
 * code is the model's reading: every verdict is marked as written by AI, and never a score. Checking
 * says what it sends and costs first, and goes out only when asked; a check is remembered while
 * neither the plan nor the code changes.
 */
export function PlanCheck(props: { go: (r: Partial<Route>) => void }) {
  const source = useSource()
  const [reload, setReload] = useState(0)
  const plan = useApi<PlanResponse>(`/api/plan${reload ? `#${reload}` : ''}`)
  const [state, setState] = useState<{ checking?: boolean; check?: Check; sent?: boolean; usd?: number; error?: string }>({})
  const [editing, setEditing] = useState(false)

  if (plan.error) return <Failure error={plan.error} />
  if (!plan.data) return <Loading q="Did the code do what I asked?" what="the plan" />
  const p = plan.data
  const check = state.check ?? p.check
  const hasPlan = p.files.length > 0

  const run = async () => {
    setState({ checking: true })
    try {
      const r = await source.act<{ check: Check; sent: boolean; usd?: number }>('/api/plan/check')
      setState({ check: r.check, sent: r.sent, usd: r.usd })
    } catch (e) {
      setState({ error: e instanceof Error ? e.message : String(e) })
    }
  }

  return (
    <div className="reading plan-check">
      <div className="page-q">Did the code do what I asked?</div>

      {!hasPlan || editing ? (
        <PlanFiles
          files={p.files}
          onSaved={() => {
            setEditing(false)
            setState({})
            setReload((n) => n + 1)
          }}
          onCancel={hasPlan ? () => setEditing(false) : undefined}
        />
      ) : p.problem ? (
        <p className="overview-answer">The plan can’t be read: {p.problem}.</p>
      ) : check ? (
        <Verdict check={check} />
      ) : (
        <>
          <p className="overview-answer">
            Your plan asks for {number(p.items.length)} {p.items.length === 1 ? 'thing' : 'things'}
            {p.more ? `, and ${p.more} more that a check leaves out` : ''}. archdoc can read each one against the code.
          </p>
          {p.earlier && <p className="muted">The last check, {day(p.checked_at)}, was for an earlier plan or earlier code. It is below, as it was.</p>}
        </>
      )}

      {hasPlan && !editing && !p.problem && !check && (
        <div className="ask-confirm plan-confirm">
          <p>
            Checking sends the {p.items.length} requirements and {p.facts} facts archdoc read — names, routes and tables, no code — to
            Anthropic, on your key. It costs {dollars(p.estimate)}.
          </p>
          <div className="ask-confirm-acts">
            <button className="btn btn-ink" onClick={run} disabled={state.checking}>
              {state.checking ? 'Reading the plan against the code…' : 'Check the plan'}
            </button>
            <button className="link-btn" onClick={() => props.go({ screen: 'runs' })}>
              What has left this machine →
            </button>
          </div>
          {state.error && (
            <p className="ask-error" role="alert">
              {aiFailure(state.error, 'The check')}
            </p>
          )}
        </div>
      )}

      {check && <Groups check={check} />}
      {!check && p.earlier && hasPlan && !editing && <Groups check={p.earlier} />}

      {hasPlan && !editing && (
        <p className="plan-foot muted small">
          {check && (state.sent ? `This check sent ${check.facts} facts · cost $${(state.usd ?? 0).toFixed(4)}. ` : `Checked ${day(p.checked_at)}; nothing was sent this time. `)}
          Plan: <span className="mono">{p.files.join(', ')}</span>{' '}
          <button className="link-btn" onClick={() => setEditing(true)}>
            Change
          </button>
        </p>
      )}
    </div>
  )
}

/** The verdict in one sentence, saying whose judgment it is (rule 14) — never a score. */
function Verdict({ check }: { check: Check }) {
  const n = (s: string) => check.verdicts.filter((v) => v.status === s).length
  const built = n('built')
  const missing = n('missing')
  const unclear = n('unclear')
  const extras = (check.extras ?? []).length
  const parts = [
    `${built} of ${check.verdicts.length} ${check.verdicts.length === 1 ? 'requirement looks' : 'requirements look'} built`,
    ...(missing ? [`${missing} ${missing === 1 ? 'looks' : 'look'} missing`] : []),
    ...(unclear ? [`${unclear} couldn’t be checked`] : []),
    ...(extras ? [`${extras} ${extras === 1 ? 'thing was' : 'things were'} built that the plan didn’t ask for`] : []),
  ]
  const last = parts.pop()
  return (
    <>
      <p className="overview-answer interp-sentence">Reading your plan against the code: {parts.length ? `${parts.join(', ')}, and ${last}` : last}.</p>
      <div className="ask-ai">
        <TruthMark state="interpreted" /> Every verdict is written by AI, with the code it matched beside it
      </div>
    </>
  )
}

const groups = [
  { status: 'built', title: 'Looks built', sub: 'with the code each matched' },
  { status: 'missing', title: 'Looks missing', sub: 'with what archdoc looked for' },
  { status: 'unclear', title: 'Couldn’t check', sub: 'too vague to check against how code is built' },
] as const

function Groups({ check }: { check: Check }) {
  return (
    <div className="plan-groups">
      {groups.map((g) => {
        const rows = check.verdicts.filter((v) => v.status === g.status)
        if (!rows.length) return null
        return (
          <section key={g.status} className={`plan-group is-${g.status}`}>
            <h2 className="meaning-head">
              {g.title} <span className="muted">· {rows.length} · {g.sub}</span>
            </h2>
            <ul>
              {rows.map((v) => (
                <li key={v.item.id}>
                  <div className="plan-asked">
                    {v.item.text} <Cite p={v.item.provenance} compact />
                  </div>
                  <Reading claim={v.claim} />
                </li>
              ))}
            </ul>
          </section>
        )
      })}
      {(check.extras ?? []).length > 0 && (
        <section className="plan-group is-extra">
          <h2 className="meaning-head">
            Built, not asked for <span className="muted">· {(check.extras ?? []).length} · in the latest version, in no requirement</span>
          </h2>
          <ul>
            {(check.extras ?? []).map((c, i) => (
              <li key={i}>
                <Reading claim={c} />
              </li>
            ))}
          </ul>
        </section>
      )}
    </div>
  )
}

function Reading({ claim }: { claim: Claim }) {
  return (
    <div className="plan-reading">
      <TruthMark state="interpreted" /> <span className="interp-sentence">{claim.text}</span>{' '}
      {(claim.cites ?? []).map((p, i) => (
        <Cite key={i} p={p} compact />
      ))}
    </div>
  )
}

/**
 * Naming the plan: the files archdoc may read as what was asked for — a spec, a task list, an
 * agent's plan (F-40). archdoc reads these and no others.
 */
function PlanFiles(props: { files: string[]; onSaved: () => void; onCancel?: () => void }) {
  const source = useSource()
  const [text, setText] = useState(props.files.join('\n'))
  const [error, setError] = useState<string>()
  const save = async () => {
    setError(undefined)
    try {
      await source.act('/api/plan/files', { files: text.split(/[\n,]+/).map((s) => s.trim()).filter(Boolean) })
      props.onSaved()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    }
  }
  return (
    <div className="plan-files">
      {!props.files.length && (
        <p className="overview-answer">No plan yet. Name the files that say what you asked for, and archdoc reads each requirement against the code.</p>
      )}
      <p className="muted">
        A spec, a task list, an agent’s plan — Markdown or text, inside this repository. archdoc reads these files and no others; each
        list item is one requirement, so a plan that works looks like:
      </p>
      <pre className="plan-example">{'- Members can comment on an album.\n- Members can delete their own comments.\n- Comments show on the album page.'}</pre>
      <label className="plan-files-label" htmlFor="plan-files">
        Plan files, one a line
      </label>
      <textarea id="plan-files" className="plan-files-input mono" rows={3} value={text} placeholder="docs/plan.md" onChange={(e) => setText(e.target.value)} />
      <div className="ask-confirm-acts">
        <button className="btn btn-ink" onClick={save} disabled={!text.trim()}>
          Read these files
        </button>
        {props.onCancel && (
          <button className="link-btn" onClick={props.onCancel}>
            Cancel
          </button>
        )}
      </div>
      {error && (
        <p className="ask-error" role="alert">
          {error}
        </p>
      )}
    </div>
  )
}
