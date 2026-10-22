import { useState } from 'react'
import type { AskAnswer, AskPreview, Claim, Estimate, Provenance } from '../api'
import { useApi } from '../api'
import type { Route } from '../route'
import { useSource } from '../source'
import { Cite } from '../ui/Cite'
import { Failure, Loading, TruthMark } from '../ui/marks'

/**
 * Why a paid request failed, in words that say what to do (platform-surface §4.8): a refused key
 * is the person's to fix, where archdoc serve was started; anything else, as the API said it.
 */
export function aiFailure(error: string, what: string): string {
  if (/credentials|401|x-api-key/i.test(error)) {
    return `${what} couldn’t be written: Anthropic refused the key. Check ANTHROPIC_API_KEY in the shell archdoc serve was started from, then try again. This try cost nothing.`
  }
  if (/credit|billing|402/i.test(error)) return `${what} couldn’t be written: the Anthropic account has no credit left. Add credit in the Anthropic console, then try again.`
  return `${what} couldn’t be written: ${error.replace(/\.$/, '')}. Everything proven is still here; this try is in What left this machine.`
}

/** A cost, roughly: "about $0.003", or "under a cent" said plainly when it is. */
export function dollars(e?: Estimate): string {
  if (!e?.known) return 'a cost archdoc can’t estimate for this model'
  if (e.usd < 0.001) return 'about a tenth of a cent'
  return `about $${e.usd < 0.01 ? e.usd.toFixed(3) : e.usd.toFixed(2)}`
}

/**
 * Ask (platform-surface §6.5): a question answered only from what archdoc read, in the right rail.
 * It says what it would send and roughly what that costs, and sends only when asked to (rule 19).
 * A question nothing matches is answered at once, for nothing; an answer asked before comes back
 * from memory. Every sentence is written by AI and marked so, and cites the facts it rests on.
 */
export function Ask(props: { question: string; go: (r: Partial<Route>) => void; onFollowUp: () => void }) {
  const source = useSource()
  const preview = useApi<AskPreview>(`/api/ask?q=${encodeURIComponent(props.question)}`)
  const [state, setState] = useState<{ asking?: boolean; answer?: AskAnswer; sent?: boolean; usd?: number; error?: string }>({})

  const ask = async () => {
    setState({ asking: true })
    try {
      const r = await source.act<{ answer: AskAnswer; sent: boolean; usd?: number }>('/api/ask/send', { q: props.question })
      setState({ answer: r.answer, sent: r.sent, usd: r.usd })
    } catch (e) {
      setState({ error: e instanceof Error ? e.message : String(e) })
    }
  }

  const p = preview.data
  const answer = state.answer ?? p?.remembered
  return (
    <div className="ask">
      <p className="ask-question">{/[?？]$/.test(props.question) ? props.question : `${props.question}?`}</p>
      {preview.error ? (
        <Failure error={preview.error} />
      ) : !p ? (
        <Loading what="what archdoc read about it" />
      ) : p.facts === 0 ? (
        <Nothing />
      ) : answer ? (
        <Answered a={answer} go={props.go} note={state.sent ? `Sent ${p.facts} facts and your question · cost $${(state.usd ?? 0).toFixed(4)}` : 'Answered before — nothing was sent this time'} />
      ) : state.asking ? (
        <p className="state-line" role="status">
          Writing from {p.facts} facts archdoc read · {dollars(p.estimate)} · estimate
        </p>
      ) : (
        <div className="ask-confirm">
          <p>
            Asking sends {p.facts} {p.facts === 1 ? 'fact' : 'facts'} archdoc read — names, routes and tables, no code — and your question to Anthropic, on
            your key. It costs {dollars(p.estimate)}.
          </p>
          <div className="ask-confirm-acts">
            <button className="btn btn-ink" onClick={ask}>
              Ask
            </button>
            <button className="link-btn" onClick={() => props.go({ screen: 'runs' })}>
              What has left this machine →
            </button>
          </div>
          {state.error && (
            <p className="ask-error" role="alert">
              {aiFailure(state.error, 'The answer')}
            </p>
          )}
        </div>
      )}
      <button className="link-btn ask-follow" onClick={props.onFollowUp}>
        Ask a follow-up…
      </button>
    </div>
  )
}

/** The designed answer when nothing archdoc read shares a word with the question. Free. */
function Nothing() {
  return (
    <div className="ask-nothing">
      <p>archdoc found nothing it can cite for this question.</p>
      <p className="muted">
        It answers only from what it read — the parts, routes, tables, jobs and the calls between them. Naming one of them usually
        helps. Nothing was sent.
      </p>
    </div>
  )
}

function Answered({ a, note, go }: { a: AskAnswer; note: string; go: (r: Partial<Route>) => void }) {
  const claims = a.claims ?? []
  const cites = numbered(claims)
  return (
    <div className="ask-answer">
      {claims.length > 0 ? (
        <>
          <div className="ask-ai">
            <TruthMark state="interpreted" /> Written by AI from these facts
          </div>
          <p className="ask-text interp-sentence">
            {claims.map((c, i) => (
              <span key={i}>
                {c.text}
                {(c.cites ?? []).map((p) => (
                  <sup key={key(p)} className="ask-ref">
                    {cites.indexOf(key(p)) + 1}
                  </sup>
                ))}{' '}
              </span>
            ))}
          </p>
        </>
      ) : (
        <p>The facts archdoc read don’t answer this.</p>
      )}
      {a.not_found && (
        <p className="ask-unresolved">
          <TruthMark state="unresolved" /> <strong>Couldn’t be resolved:</strong> {a.not_found}
        </p>
      )}
      {cites.length > 0 && (
        <ol className="ask-cites">
          {claims
            .flatMap((c) => c.cites ?? [])
            .filter((p, i, all) => all.findIndex((x) => key(x) === key(p)) === i)
            .map((p) => (
              <li key={key(p)}>
                <Cite p={p} />
              </li>
            ))}
        </ol>
      )}
      {note && (
        <p className="ask-note muted small">
          {note}{' '}
          <button className="link-btn" onClick={() => go({ screen: 'runs' })}>
            What left this machine →
          </button>
        </p>
      )}
    </div>
  )
}

const key = (p: Provenance) => `${p.file}:${p.line}`

/** The answer's citations, numbered once each in the order they first appear. */
function numbered(claims: Claim[]): string[] {
  const out: string[] = []
  for (const c of claims) for (const p of c.cites ?? []) if (!out.includes(key(p))) out.push(key(p))
  return out
}

