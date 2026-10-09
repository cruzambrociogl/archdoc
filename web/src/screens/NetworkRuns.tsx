import type { Exchange, Run } from '../api'
import { bytes, published, useApi, when } from '../api'
import type { Route } from '../route'
import { Planned } from '../ui/Planned'
import { Eyebrow, Failure, Loading } from '../ui/marks'

/**
 * What left the machine (surface-spec §5.14). Every run that used the network, and each request
 * exactly as it was sent — unformatted on purpose, because a prettier version would no longer be
 * the thing that was sent. The empty state is a feature, and says so.
 */
export function NetworkRuns(props: { route: Route; go: (r: Partial<Route>, o?: { replace?: boolean }) => void }) {
  const runs = useApi<Run[]>('/api/runs')
  if (runs.error) return <Failure error={runs.error} />
  if (!runs.data) return <Loading />
  const list = runs.data

  if (!list.length) {
    return (
      <div className="reading">
        <Eyebrow>Network runs · what left the machine</Eyebrow>
        <p className="statement">archdoc has never sent anything from this repository.</p>
        <p className="lede-quiet">
          Every run so far was offline. archdoc calls out only when asked to, with <span className="mono">archdoc label</span> or <span className="mono">archdoc explain</span>,
          and when it does, each request appears here exactly as it went out — including the ones that failed.
        </p>
      </div>
    )
  }

  const sent = list.reduce((s, r) => s + r.bytes_sent, 0)
  const cost = list.reduce((s, r) => s + (r.cost_known ? r.cost_usd : 0), 0)
  const open = props.route.run ?? list[0].id

  return (
    <div className="reading wide">
      <Eyebrow>Network runs · what left the machine</Eyebrow>
      <h1 className="page-title small">
        {list.length} {list.length === 1 ? 'run' : 'runs'} · {bytes(sent)} sent
      </h1>
      <p className="lede-quiet">${cost.toFixed(4)} where the model's price is known. Every byte below is what the server received.</p>

      <div className="runs-table">
        <div className="runs-row runs-head">
          <span>When</span>
          <span>Model</span>
          <span>Egress</span>
          <span>Requests</span>
          <span>Bytes</span>
          <span>Tokens in / out</span>
          <span>Cost</span>
          <span>Status</span>
        </div>
        {list.map((r) => (
          <button key={r.id} className={`runs-row ${r.id === open ? 'on' : ''}`} onClick={() => props.go({ screen: 'runs', run: r.id }, { replace: true })}>
            <span>{when(r.started_at)}</span>
            <span>{r.model || '—'}</span>
            <span>{r.egress_mode}</span>
            <span>{r.requests}</span>
            <span>{bytes(r.bytes_sent)}</span>
            <span>
              {r.tokens_in.toLocaleString()} / {r.tokens_out.toLocaleString()}
            </span>
            <span>{r.cost_known ? `$${r.cost_usd.toFixed(4)}` : 'unknown'}</span>
            <span className={r.status === 'ok' ? '' : 'strong'}>{r.status}</span>
          </button>
        ))}
      </div>

      {published ? (
        <p className="lede-quiet">
          This published site carries only the summary above. Each request, exactly as it was sent, stays on the machine that
          made it, in archdoc serve.
        </p>
      ) : (
        <RunDetail id={open} go={props.go} />
      )}
    </div>
  )
}

// What each egress mode sends; one archdoc does not know yet is shown by its name.
const modes: Record<string, string> = {
  'structure-only': 'names, kinds and relationships only — no file contents, literals or comments',
  'structure-and-summaries':
    "names, paths, counts, routes and table columns, plus each route's own one-line summary as the code states it — no code, no other text from a file, no line numbers",
}

function RunDetail({ id, go }: { id: number; go: (r: Partial<Route>) => void }) {
  const run = useApi<Run & { exchanges: Exchange[] | null }>(`/api/runs/${id}`)
  if (run.error) return <Failure error={run.error} />
  if (!run.data) return <Loading />
  const r = run.data
  const ex = r.exchanges ?? []

  return (
    <div className="run-detail">
      <div className="run-plain">
        <Eyebrow>In plain words · run of {when(r.started_at)}</Eyebrow>
        <p className="run-plain-text">
          {r.requests} {r.requests === 1 ? 'request' : 'requests'} to {r.model || 'the model'}, {bytes(r.bytes_sent)} in all:{' '}
          {modes[r.egress_mode] ?? r.egress_mode}. {r.status === 'ok' ? 'It completed.' : `It ended ${r.status}.`}
        </p>
      </div>
      {ex.map((x, i) => (
        <div key={i} className="exchange">
          <div className="exchange-head">
            <Eyebrow>
              Request {i + 1} · exactly as sent · {bytes(x.body.length)}
            </Eyebrow>
            <span className="mono small muted">
              {x.method} {x.url} → {x.status || 'no response'}{' '}
              {i === 0 && (
                <Planned id="run-download" go={go}>
                  Copy · .json
                </Planned>
              )}
            </span>
          </div>
          <pre className="payload">{x.body}</pre>
          {x.response && (
            <details className="files">
              <summary>
                <Eyebrow>What came back · {bytes(x.response.length)} · kept on this machine</Eyebrow>
              </summary>
              <pre className="payload">{x.response}</pre>
            </details>
          )}
        </div>
      ))}
      <p className="lede-quiet">Shown unformatted on purpose: a prettier version would no longer be the thing that was sent. The API key is never stored.</p>
    </div>
  )
}
