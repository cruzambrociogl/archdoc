import { useEffect, useState } from 'react'
import type { UpdateAsk, UpdatePreview } from '../api'
import type { Route } from '../route'
import { dollars } from '../screens/Ask'
import { number } from '../sentence'
import { useSource } from '../source'
import { TruthMark } from '../ui/marks'
import { lookNow } from './updating'

const message = (e: unknown) => (e instanceof Error ? e.message : String(e))

/**
 * Update (platform-surface §4.3): read the repository again and regenerate its documentation — free,
 * offline, one click, as archdoc generate does. The run is followed in the notice band.
 */
export function UpdateButton(props: { busy: boolean }) {
  const source = useSource()
  const [state, setState] = useState<{ asking?: boolean; error?: string }>({})
  return (
    <>
      <button
        className="btn btn-ink"
        disabled={state.asking || props.busy}
        onClick={async () => {
          setState({ asking: true })
          try {
            await source.act('/api/update')
            lookNow(source)
            setState({})
          } catch (e) {
            setState({ error: message(e) })
          }
        }}
      >
        {props.busy ? 'Updating…' : 'Update'}
      </button>
      {state.error && (
        <p className="pop-note" role="alert">
          {state.error}
        </p>
      )}
    </>
  )
}

// What each paid run is, in the words the person chooses by.
const runs: Record<UpdateAsk['what'], { title: string; one: string; many: string }> = {
  label: { title: 'Descriptions and relationship labels', one: 'one request for the whole system', many: 'requests for the whole system' },
  explain: { title: 'What each part does, every sentence cited', one: 'one request, for one part', many: 'requests, one per part' },
}

const size = (bytes: number) => (bytes < 1024 ? `${bytes} bytes` : `${Math.round(bytes / 1024)} KB`)

/**
 * Update with AI (§4.3, rule 19): a separate, deliberate step. First what each paid run would send
 * and roughly what it costs — from a dry run of the pipeline, so the figures are the engine's — and
 * only then, on the person's say, the update. The server sends nothing beyond what was shown here.
 */
export function UpdateWithAI(props: { busy: boolean; go: (r: Partial<Route>) => void; onStarted: () => void }) {
  const source = useSource()
  const [preview, setPreview] = useState<{ data?: UpdatePreview; error?: string }>({})
  const [off, setOff] = useState<Partial<Record<UpdateAsk['what'], boolean>>>({})
  const [start, setStart] = useState<{ asking?: boolean; error?: string }>({})

  useEffect(() => {
    let live = true
    source.act<UpdatePreview>('/api/update/preview', { label: true, explain: true }).then(
      (data) => live && setPreview({ data }),
      (e) => live && setPreview({ error: message(e) }),
    )
    return () => {
      live = false
    }
  }, [source])

  if (preview.error) return <p className="pop-note" role="alert">{`What it would send couldn’t be worked out: ${preview.error}`}</p>
  if (!preview.data)
    return (
      <p className="pop-note" role="status">
        Working out what it would send — archdoc reads the repository first, free and offline…
      </p>
    )

  const { asks, key } = preview.data
  const chosen = asks.filter((a) => !off[a.what])
  const explains = asks.some((a) => a.what === 'explain')
  const total = { usd: chosen.reduce((n, a) => n + a.usd, 0), known: chosen.every((a) => a.known) }

  const send = async () => {
    setStart({ asking: true })
    try {
      const agreed = Object.fromEntries(chosen.map((a) => [a.what, a]))
      await source.act('/api/update', { label: !!agreed.label, explain: !!agreed.explain, agreed })
      lookNow(source)
      props.onStarted()
    } catch (e) {
      setStart({ error: message(e) })
    }
  }

  return (
    <div className="update-ai">
      <p className="update-ai-lead">
        <TruthMark state="interpreted" /> Sends what archdoc read — names, paths and the code’s own summaries, no code — to Anthropic, on your key. Choose what to write:
      </p>
      {asks.map((a) => (
        <label key={a.what} className="update-ai-run">
          <input type="checkbox" checked={!off[a.what]} onChange={(e) => setOff((o) => ({ ...o, [a.what]: !e.target.checked }))} />
          <span>
            <span className="update-ai-title">{runs[a.what].title}</span>
            <span className="update-ai-sub">
              {a.requests === 1 ? runs[a.what].one : `${number(a.requests)} ${runs[a.what].many}`} · {size(a.bytes)} · {dollars(a)}
            </span>
          </span>
        </label>
      ))}
      {!explains && <p className="update-ai-sub">Nothing new to explain: every part already has an explanation for what it is now, or is too small to ask about.</p>}
      {key ? (
        <p className="update-ai-sub">
          {chosen.length ? (
            <>
              In all {dollars(total)}, estimated from its size, to {asks[0]?.model}. A corrected answer is asked for again and costs a little more; never more requests than these.
            </>
          ) : (
            'Choose at least one.'
          )}
        </p>
      ) : (
        <p className="pop-note" role="alert">
          ANTHROPIC_API_KEY isn’t set where archdoc serve was started. Set it in that shell, start archdoc serve again, and come back. Nothing has been sent.
        </p>
      )}
      <div className="pop-actions">
        <button className="btn btn-ink" disabled={!key || !chosen.length || start.asking || props.busy} onClick={send}>
          {props.busy ? 'Updating…' : 'Update with AI'}
        </button>
        <button className="link-btn" onClick={() => props.go({ screen: 'runs' })}>
          What has left this machine →
        </button>
      </div>
      {start.error && (
        <p className="pop-note" role="alert">
          {start.error}
        </p>
      )}
    </div>
  )
}
