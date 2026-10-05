import type { CoverageResponse, DiffResponse, ModelResponse, Node, Provenance, Run, Summary, Version } from '../api'
import { bytes, componentLevel, useApi } from '../api'
import type { Route } from '../route'
import { Cite } from '../ui/Cite'
import { kindStyle } from '../ui/kinds'
import { Eyebrow, Failure, Loading, TruthMark } from '../ui/marks'

/**
 * The landing page (surface-spec §5.1): what this system is, at a glance, and how much of it can
 * be trusted. Everything here is counted from what the engine stored; nothing is estimated.
 */
export function Overview(props: { summary: Summary; versions: Version[]; version: number | null; go: (r: Partial<Route>) => void; coverage?: CoverageResponse }) {
  const q = props.version ? `?version=${props.version}` : ''
  const m = useApi<ModelResponse>(`/api/model${q}`)
  const runs = useApi<Run[]>('/api/runs')
  const current = m.data?.version
  const previous = [...props.versions].sort((a, b) => b.id - a.id).find((v) => current !== undefined && v.id < current)?.id
  const diff = useApi<DiffResponse>(current && previous ? `/api/diff?from=${previous}&to=${current}` : null)

  if (m.error) return <Failure error={m.error} />
  if (!m.data) return <Loading />
  const { model, container, context } = m.data
  // Configuration's view of the system; the components read from code are counted on their own.
  const parts = new Set((model.nodes ?? []).filter((n) => n.kind === 'component').map((n) => n.id))
  const nodes = (model.nodes ?? []).filter((n) => !parts.has(n.id))
  const edges = (model.edges ?? []).filter((e) => !parts.has(e.from) && !parts.has(e.to))
  const system = (context.nodes ?? []).find((n) => n.kind === 'system')

  const count = (pred: (n: Node) => boolean) => nodes.filter(pred).length
  // The engine's own rule (archdoc.Kind.Container): applications, data stores and queues.
  const containers = count((n) => ['application', 'datastore', 'queue'].includes(n.kind))
  const stores = count((n) => n.kind === 'datastore' || n.kind === 'queue')
  const externals = count((n) => n.kind === 'external')

  // Every value on screen, and how many of them the model wrote: the interpreted share.
  const values: (Provenance | undefined)[] = [
    ...nodes.flatMap((n) => [
      n.name_provenance,
      ...(n.description ? [n.description_provenance] : []),
      ...(n.technology ? [n.technology_provenance] : []),
    ]),
    ...edges.filter((e) => e.label).map((e) => e.label_provenance),
  ]
  const shown = values.length
  const written = values.filter((p) => p?.origin === 'model').length
  const interpretedPct = shown ? Math.round((written / shown) * 100) : 0

  const sent = (runs.data ?? []).reduce((s, r) => s + r.bytes_sent, 0)
  const runCount = runs.data?.length ?? 0

  const when = new Date(m.data.created_at)
  const summary = system?.description

  return (
    <div className="reading wide">
      <Eyebrow>
        Overview · generated from {m.data.commit ? m.data.commit.slice(0, 7) : 'an uncommitted tree'} on{' '}
        {when.toLocaleDateString(undefined, { day: 'numeric', month: 'short', year: 'numeric' })}
      </Eyebrow>
      <h1 className="page-title">{props.summary.name}</h1>

      {summary ? (
        <>
          <p className="lede">
            <span className={system?.description_provenance?.origin === 'model' ? 'interp-sentence' : ''}>{summary}</span>
          </p>
          {system?.description_provenance && (
            <p className="lede-note">
              {system.description_provenance.origin === 'model' && <TruthMark state="interpreted" />}{' '}
              {system.description_provenance.origin === 'model' ? 'Interpreted by the model' : 'From the repository'} ·{' '}
              <Cite p={system.description_provenance} compact />
            </p>
          )}
        </>
      ) : (
        <p className="lede">
          {containers} {containers === 1 ? 'container' : 'containers'}, {edges.length} relationships and {externals}{' '}
          external {externals === 1 ? 'system' : 'systems'}, read from <span className="mono">{model.source}</span>. No
          description has been written for it — run <span className="mono">archdoc generate --label</span> for one, cited like
          everything else.
        </p>
      )}

      <div className="stat-strip">
        <Stat n={containers} label="containers" onClick={() => props.go({ screen: 'explorer', level: 'container' })} />
        <Stat n={stores} label="data stores and queues" onClick={() => props.go({ screen: 'explorer', level: 'container' })} />
        <Stat n={externals} label="external systems" onClick={() => props.go({ screen: 'explorer', level: 'context' })} />
        <Stat n={edges.length} label="relationships" onClick={() => props.go({ screen: 'explorer', level: 'container' })} />
        {parts.size > 0 && (
          <Stat
            n={parts.size}
            label={`components in ${m.data.components.length} ${m.data.components.length === 1 ? 'container' : 'containers'}, from the code`}
            onClick={() => props.go({ screen: 'explorer', level: componentLevel(m.data!.components[0]) })}
          />
        )}
        <Stat n={props.summary.versions} label={props.summary.versions === 1 ? 'version' : 'versions'} onClick={() => props.go({ screen: 'changes' })} />
      </div>

      <div className="trust-strip">
        <Trust
          k="Interpreted"
          v={`${interpretedPct}%`}
          pct={interpretedPct}
          of={written ? `${written} of ${shown} values were written by the model; each is marked and cited.` : 'Every value on screen was read from the repository.'}
        />
        <Trust
          k="Left this machine"
          v={runCount ? bytes(sent) : 'nothing'}
          pct={0}
          of={runCount ? `${runCount} ${runCount === 1 ? 'run' : 'runs'}, every byte logged in Network runs.` : 'archdoc has never sent anything from this repository.'}
          onClick={() => props.go({ screen: 'runs' })}
        />
        {props.coverage ? (
          <Trust
            k="Could not be resolved"
            v={`${props.coverage.gaps.reduce((n, g) => n + g.gaps.length, 0)} gaps`}
            pct={props.coverage.items ? Math.round((props.coverage.complete / props.coverage.items) * 100) : 0}
            of={`${props.coverage.complete} of ${props.coverage.items} elements and relationships carry no gap. Coverage lists each one.`}
            onClick={() => props.go({ screen: 'coverage' })}
          />
        ) : (
          <Trust k="Evidence" v={model.source} pct={100} of="Every element and relationship cites the file and line that declares it." mono />
        )}
      </div>

      <div className="overview-grid">
        <section>
          <div className="section-head">
            <Eyebrow>Containers</Eyebrow>
            <button className="text-link" onClick={() => props.go({ screen: 'explorer', level: 'container' })}>
              Open in explorer
            </button>
          </div>
          <button className="thumb" onClick={() => props.go({ screen: 'explorer', level: 'container' })}>
            <div className="thumb-boundary">
              {(container.nodes ?? [])
                .filter((n) => n.kind !== 'actor')
                .slice(0, 9)
                .map((n) => (
                  <div key={n.id} className={`thumb-box tile-${kindStyle(n.kind).hue}`}>
                    {n.name}
                  </div>
                ))}
            </div>
            <div className="thumb-foot">
              <span className="mono muted small">click any box to inspect it</span>
              <span className="btn btn-ink">Open explorer →</span>
            </div>
          </button>
        </section>

        <section>
          <div className="section-head">
            <Eyebrow>What changed{previous ? ` · since v${previous}` : ''}</Eyebrow>
            {previous && (
              <button className="text-link" onClick={() => props.go({ screen: 'changes' })}>
                Changes
              </button>
            )}
          </div>
          {!previous && <p className="muted small">This is the first version; there is nothing to compare it with yet.</p>}
          {diff.data && <ChangeList d={diff.data} />}
        </section>
      </div>

      <Eyebrow>Where to start</Eyebrow>
      <div className="paths">
        <PathCard t="Understand it from the top" route="context → containers → each element's passport" onClick={() => props.go({ screen: 'explorer', level: 'context' })} />
        <PathCard t="What changed" route="the versions, and the difference between any two" onClick={() => props.go({ screen: 'changes' })} />
        <PathCard
          t="Check the evidence"
          route={props.coverage ? 'what archdoc could not see, and why' : 'the documents, and what archdoc could not see'}
          onClick={() => props.go({ screen: props.coverage ? 'coverage' : 'docs' })}
        />
      </div>
    </div>
  )
}

function Stat({ n, label, onClick }: { n: number; label: string; onClick: () => void }) {
  return (
    <button className="stat" onClick={onClick}>
      <span className="stat-n">{n}</span>
      <span className="stat-label">{label}</span>
    </button>
  )
}

function Trust(props: { k: string; v: string; pct: number; of: string; onClick?: () => void; mono?: boolean }) {
  const body = (
    <>
      <Eyebrow>{props.k}</Eyebrow>
      <div className={`trust-v ${props.mono ? 'trust-v-small' : ''}`}>{props.v}</div>
      <div className="trust-bar">
        <div style={{ width: `${props.pct}%` }} />
      </div>
      <div className="trust-of">{props.of}</div>
    </>
  )
  return props.onClick ? (
    <button className="trust" onClick={props.onClick}>
      {body}
    </button>
  ) : (
    <div className="trust">{body}</div>
  )
}

function ChangeList({ d }: { d: DiffResponse }) {
  if (d.empty) return <p className="muted small">Nothing changed between v{d.from} and v{d.to}.</p>
  const x = d.diff
  const rows: [string, string][] = [
    ...(x.added_nodes ?? []).map((n): [string, string] => ['+', `${n.name} appeared`]),
    ...(x.removed_nodes ?? []).map((n): [string, string] => ['−', `${n.name} disappeared`]),
    ...(x.added_edges ?? []).map((e): [string, string] => ['+', `${e.from} → ${e.to}`]),
    ...(x.removed_edges ?? []).map((e): [string, string] => ['−', `${e.from} → ${e.to}`]),
    ...(x.changed ?? []).map((c): [string, string] => ['±', `${c.element}: ${c.field} changed`]),
  ]
  return (
    <ul className="change-list">
      {rows.slice(0, 6).map(([g, t], i) => (
        <li key={i}>
          <span className={`delta-tag ${d.structural ? '' : 'hollow'}`}>{g}</span>
          <span>{t}</span>
        </li>
      ))}
      {rows.length > 6 && <li className="muted small">and {rows.length - 6} more</li>}
    </ul>
  )
}

function PathCard({ t, route, onClick }: { t: string; route: string; onClick: () => void }) {
  return (
    <button className="path-card" onClick={onClick}>
      <span className="path-title">{t}</span>
      <span className="path-route">{route}</span>
    </button>
  )
}
