import { useState } from 'react'
import type { Exchange, Run } from './api'
import { bytes, useApi, when } from './api'
import { Failure, Loading } from './Prov'

/** Every run that used the network, and the exact bytes it sent. */
export function Runs() {
  const runs = useApi<Run[]>('/api/runs')
  const [open, setOpen] = useState<number | null>(null)
  if (runs.error) return <Failure error={runs.error} />
  if (!runs.data) return <Loading />

  if (!runs.data.length) {
    return (
      <>
        <h2>Network runs</h2>
        <p>
          No run has used the network. archdoc only calls out when asked to, with <code>generate --label</code>, and
          records every such run here — including the ones that failed.
        </p>
      </>
    )
  }

  const total = runs.data.reduce((sum, r) => sum + (r.cost_known ? r.cost_usd : 0), 0)

  return (
    <>
      <div className="toolbar">
        <h2>Network runs</h2>
        <span className="muted small">
          {runs.data.length} runs · ${total.toFixed(4)} where the price is known
        </span>
      </div>
      <table>
        <thead>
          <tr>
            <th>#</th>
            <th>Started</th>
            <th>Status</th>
            <th>Model</th>
            <th>Sent</th>
            <th>Requests</th>
            <th>Tokens in / out</th>
            <th>Cost</th>
          </tr>
        </thead>
        <tbody>
          {runs.data.map((r) => (
            <tr key={r.id} className={open === r.id ? 'active' : ''} onClick={() => setOpen(open === r.id ? null : r.id)}>
              <td>{r.id}</td>
              <td className="small">{when(r.started_at)}</td>
              <td>
                <span className={`state ${r.status === 'ok' ? 'written' : 'missing'}`}>{r.status}</span>
              </td>
              <td className="mono small">{r.model}</td>
              <td>{bytes(r.bytes_sent)}</td>
              <td>{r.requests}</td>
              <td>
                {r.tokens_in.toLocaleString()} / {r.tokens_out.toLocaleString()}
              </td>
              <td>{r.cost_known ? `$${r.cost_usd.toFixed(4)}` : <span className="muted">unknown</span>}</td>
            </tr>
          ))}
        </tbody>
      </table>
      {open !== null && <RunDetail id={open} />}
    </>
  )
}

function RunDetail({ id }: { id: number }) {
  const run = useApi<Run & { exchanges: Exchange[] | null }>(`/api/runs/${id}`)
  if (run.error) return <Failure error={run.error} />
  if (!run.data) return <Loading />
  const ex = run.data.exchanges ?? []
  return (
    <>
      <h3>
        Run {id}: what left this machine ({run.data.egress_mode})
      </h3>
      <p className="muted small">Shown exactly as sent — the request bodies are not reformatted.</p>
      {ex.map((x, i) => (
        <div key={i} className="exchange">
          <div className="mono small">
            {x.method} {x.url} → {x.status || 'no response'} · {bytes(x.body.length)}
          </div>
          <pre>{x.body}</pre>
        </div>
      ))}
    </>
  )
}
