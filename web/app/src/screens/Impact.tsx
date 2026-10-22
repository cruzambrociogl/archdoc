import type { Entry, Flow, ModelResponse, Provenance } from '../api'
import { useApi } from '../api'
import type { Route } from '../route'
import { Cite } from '../ui/Cite'
import { Failure, Loading } from '../ui/marks'

/**
 * Impact (platform-surface §6.4): what does this touch? Built from the flows the engine followed —
 * every flow a part or table takes part in. The sentence is deterministic: it names the flows, then
 * the jobs, each in alphabetical order, the first five and a count of the rest; it never picks which
 * to mention. Below it, who uses the element, what it uses, the routes and jobs through it and the
 * outside services it reaches — each item cited, each opening where it lives.
 */
export function Impact(props: { element: string; version: number | null; go: (r: Partial<Route>) => void }) {
  const q = props.version ? `?version=${props.version}` : ''
  const m = useApi<ModelResponse>(`/api/model${q}`)
  if (m.error) return <Failure error={m.error} />
  if (!m.data) return <Loading />
  const model = m.data.model
  const nodes = new Map((model.nodes ?? []).map((n) => [n.id, n]))
  const node = nodes.get(props.element)
  const name = (id?: string) => (id ? (nodes.get(id)?.name ?? id) : '')
  const x = props.element
  const entries = new Map((model.entries ?? []).map((e) => [e.id, e]))

  // A flow touches the element when one of its participants is the element, or belongs to it.
  const isX = (p: Flow['participants'][number]) => p.element === x || p.component === x
  const ofContainer = (e?: Entry) => !!e && e.container === x
  const touching = (model.flows ?? []).filter((f) => f.participants.some(isX) || ofContainer(entries.get(f.entry)))
  const through = touching.map((f) => entries.get(f.entry)).filter((e): e is Entry => !!e)
  const label = (e: Entry) => (e.kind === 'job' || e.kind === 'command' ? e.path || e.handler : e.summary ? e.summary.charAt(0).toLowerCase() + e.summary.slice(1) : `${e.kind === 'page' ? 'page' : e.method} ${e.path}`)
  const flows = through.filter((e) => e.kind !== 'job').sort((a, b) => label(a).localeCompare(label(b)))
  const jobs = through.filter((e) => e.kind === 'job').sort((a, b) => label(a).localeCompare(label(b)))

  // Who calls into it, and what it calls, from the steps of those flows.
  const users = new Map<string, Provenance>()
  const uses = new Map<string, Provenance>()
  for (const f of touching) {
    const ps = new Map(f.participants.map((p) => [p.id, p]))
    for (const s of f.steps) {
      const from = ps.get(s.from)
      const to = ps.get(s.to)
      if (!from || !to) continue
      const fromId = from.element ?? from.component
      const toId = to.element ?? to.component
      if (to && isX(to) && fromId && fromId !== x && !users.has(fromId)) users.set(fromId, s.provenance)
      if (from && isX(from) && toId && toId !== x && !uses.has(toId)) uses.set(toId, s.provenance)
    }
  }
  // The outside services it — or, for a part inside an app, its app — reaches.
  const reach = (model.edges ?? []).filter((e) => (e.from === x || e.from === node?.parent) && nodes.get(e.to)?.kind === 'external')

  const named = [...flows, ...jobs].map(label)
  const said = named.length <= 5 ? join(named) : `${named.slice(0, 5).join(', ')} and ${named.length - 5} more`
  const counts = [flows.length ? n(flows.length, 'flow', 'flows') : '', jobs.length ? n(jobs.length, 'job', 'jobs') : ''].filter(Boolean)

  const groups: { title: string; items: { key: string; text: string; cite?: Provenance; on?: () => void }[] }[] = [
    { title: 'Routes and pages through it', items: flows.map((e) => ({ key: e.id, text: `${e.kind === 'page' ? 'page' : e.method} ${e.path}`, cite: e.provenance, on: () => props.go({ screen: 'entries', focus: e.id }) })) },
    { title: 'Background jobs', items: jobs.map((e) => ({ key: e.id, text: label(e), cite: e.provenance, on: () => props.go({ screen: 'entries', focus: e.id }) })) },
    { title: 'Who uses it', items: [...users].map(([id, p]) => ({ key: id, text: name(id), cite: p, on: () => props.go({ screen: 'explorer', focus: id }) })) },
    { title: 'What it uses', items: [...uses].map(([id, p]) => ({ key: id, text: name(id), cite: p, on: () => props.go({ screen: 'explorer', focus: id }) })) },
    { title: 'Outside services it reaches', items: reach.map((e) => ({ key: e.to, text: name(e.to), cite: (e.provenance ?? [])[0] })) },
  ].filter((g) => g.items.length)

  return (
    <div className="impact">
      <p className="impact-sentence">
        Changing <span className="mono">{node?.name ?? x}</span>{' '}
        {counts.length ? (
          <>
            affects {counts.join(' and ')}: {said}.
          </>
        ) : (
          <>affects no flow archdoc followed through the code. A flow it is part of may start somewhere archdoc doesn't read.</>
        )}
      </p>
      <p className="overview-source">Written from the flows archdoc followed — proven, and complete only as far as they go.</p>
      {groups.map((g) => (
        <section key={g.title} className="impact-group">
          <div className="impact-group-head">
            <span>{g.title}</span>
            <span className="muted">{g.items.length}</span>
          </div>
          {g.items.slice(0, 5).map((it) => (
            <div key={it.key} className="impact-row">
              {it.on ? (
                <button className="part-link" onClick={it.on}>
                  {it.text}
                </button>
              ) : (
                <span>{it.text}</span>
              )}
              {it.cite?.file && <Cite p={it.cite} compact />}
            </div>
          ))}
          {g.items.length > 5 && <div className="impact-more muted">and {g.items.length - 5} more</div>}
        </section>
      ))}
    </div>
  )
}

const n = (count: number, one: string, many: string) => `${count} ${count === 1 ? one : many}`
function join(xs: string[]): string {
  return xs.length <= 1 ? (xs[0] ?? '') : `${xs.slice(0, -1).join(', ')} and ${xs[xs.length - 1]}`
}

