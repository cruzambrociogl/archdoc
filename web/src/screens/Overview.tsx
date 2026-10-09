import type { CoverageResponse, DiffResponse, ModelResponse, Node, Provenance, Run, Summary, Version } from '../api'
import { bytes, componentLevel, dataLevel, useApi } from '../api'
import type { Route } from '../route'
import { Cite } from '../ui/Cite'
import { Planned } from '../ui/Planned'
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
  const tables = (model.nodes ?? []).filter((n) => n.kind === 'table')
  const tableIds = new Set(tables.map((n) => n.id))
  const nodes = (model.nodes ?? []).filter((n) => !parts.has(n.id) && !tableIds.has(n.id))
  const edges = (model.edges ?? []).filter((e) => !parts.has(e.from) && !parts.has(e.to) && !tableIds.has(e.from))
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

  const applications = count((n) => n.kind === 'application')
  const flows = (model.flows ?? []).length
  const ways = (model.entries ?? []).length
  const unresolved = (model.unresolved ?? []).length
  const coveragePct = props.coverage?.items ? Math.floor((props.coverage.complete / props.coverage.items) * 100) : 0
  const name = (id: string) => (model.nodes ?? []).find((n) => n.id === id)?.name ?? id
  const kindOf = (id: string) => (model.nodes ?? []).find((n) => n.id === id)?.kind
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

      {props.summary.tiny && props.summary.story ? (
        // A small project is told, not counted: what it is, how it is used, what it reaches.
        <div className="story">
          {props.summary.story.map((l, i) => (
            <p key={i} className={i === 0 ? 'lede' : 'story-line'}>
              {l.interpreted && <TruthMark state="interpreted" />}{' '}
              <span className={l.interpreted ? 'interp-sentence' : ''}>{l.text.split('`').map((part, j) => (j % 2 ? <span key={j} className="mono">{part}</span> : part))}</span>{' '}
              {l.provenance?.file && <Cite p={l.provenance} compact />}
            </p>
          ))}
        </div>
      ) : summary ? (
        <>
          <p className="lede">
            <span className={system?.description_provenance?.origin === 'model' ? 'interp-sentence' : ''}>{summary}</span>
          </p>
          {system?.description_provenance && (
            <p className="lede-note">
              {system.description_provenance.origin === 'model' && <TruthMark state="interpreted" />}{' '}
              {system.description_provenance.origin === 'model' ? 'Interpreted by the model' : 'From the repository'} ·{' '}
              <Cite p={system.description_provenance} compact />{' '}
              <Planned id="plain" go={props.go}>
                Plain language
              </Planned>
            </p>
          )}
        </>
      ) : (
        <p className="lede">
          {containers} {containers === 1 ? 'container' : 'containers'}, {edges.length} relationships and {externals}{' '}
          external {externals === 1 ? 'system' : 'systems'}, read from <span className="mono">{model.source}</span>. No
          description has been written for it — run <span className="mono">archdoc label</span> for one, cited like
          everything else.
        </p>
      )}

      {/* The design's strip (Surface Screens): what there is, at each level, each a way in. */}
      <div className="stat-strip">
        {!props.summary.tiny && <Stat n={applications} label="applications" onClick={() => props.go({ screen: 'explorer', level: 'container' })} />}
        {!props.summary.tiny && <Stat n={containers} label="containers" onClick={() => props.go({ screen: 'explorer', level: 'container' })} />}
        {parts.size > 0 && (
          <Stat n={parts.size} label="components" onClick={() => props.go({ screen: 'explorer', level: componentLevel(m.data!.components[0]) })} />
        )}
        {tables.length > 0 && <Stat n={tables.length} label="entities" onClick={() => props.go({ screen: 'explorer', level: dataLevel(tables[0].parent!) })} />}
        {flows > 0 && <Stat n={flows} label="flows" onClick={() => props.go({ screen: 'features' })} />}
        {ways > 0 && <Stat n={ways} label="entry points" onClick={() => props.go({ screen: 'features' })} />}
        <Stat n={externals} label={externals === 1 ? 'external system' : 'external systems'} onClick={() => props.go({ screen: 'explorer', level: 'context' })} />
      </div>

      <div className="trust-strip four">
        {props.coverage ? (
          <Trust
            k="Coverage"
            v={`${coveragePct}%`}
            pct={coveragePct}
            of={`${props.coverage.complete} of ${props.coverage.items} elements and relationships carry no gap`}
            onClick={() => props.go({ screen: 'coverage' })}
          />
        ) : (
          <Trust k="Evidence" v={model.source} pct={100} of="Every element and relationship cites the file and line that declares it." mono />
        )}
        <Trust k="Unresolved" v={String(unresolved)} pct={0} tone="unres" of={unresolved ? 'seen and listed, not dropped' : 'every call the code makes goes somewhere it names'} onClick={() => props.go({ screen: 'coverage' })} />
        <Trust
          k="Interpreted"
          v={`${interpretedPct}%`}
          pct={interpretedPct}
          tone="interp"
          of={written ? 'of visible values written by the model, each marked and cited' : 'every value on screen was read from the repository'}
        />
        <Trust
          k="Egress"
          v={runCount ? bytes(sent) : 'nothing'}
          pct={0}
          of={runCount ? `${runCount} ${runCount === 1 ? 'run' : 'runs'} · every byte logged · no code` : 'nothing has ever left this machine'}
          onClick={() => props.go({ screen: 'runs' })}
        />
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
                .filter((n) => ['application', 'datastore', 'queue'].includes(n.kind))
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
          {diff.data && <ChangeList d={diff.data} name={name} kind={kindOf} />}
        </section>
      </div>

      <Eyebrow>Where to start</Eyebrow>
      <div className="paths">
        <PathCard
          t="Understand it from the top"
          route={
            props.summary.tiny
              ? 'what it reaches → what it is made of'
              : parts.size > 0
                ? `containers → components${flows ? ' → its flows' : ''}`
                : "context → containers → each element's passport"
          }
          onClick={() => props.go({ screen: 'explorer', level: props.summary.tiny ? 'context' : 'container' })}
        />
        <PathCard
          t="See what changed"
          route={
            diff.data && previous
              ? `v${previous} → v${current} · ${structuralOf(diff.data)} structural · ${wordsOf(diff.data)} words`
              : 'the first version · nothing to compare yet'
          }
          onClick={() => props.go({ screen: 'changes' })}
        />
        <PathCard
          t="Check the evidence"
          route={props.coverage ? `coverage · ${props.coverage.gaps.reduce((n, g) => n + g.gaps.length, 0)} gaps · ${unresolved} unresolved` : 'the documents, and what archdoc could not see'}
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

function Trust(props: { k: string; v: string; pct: number; of: string; onClick?: () => void; mono?: boolean; tone?: 'unres' | 'interp' }) {
  const body = (
    <>
      <Eyebrow>{props.k}</Eyebrow>
      <div className={`trust-v ${props.mono ? 'trust-v-small' : ''} ${props.tone ? `tone-${props.tone}` : ''}`}>{props.v}</div>
      <div className={`trust-bar ${props.tone ? `tone-${props.tone}` : ''}`}>
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

const part = (k?: string) => k === 'component' || k === 'table' || k === 'module'
const wordFields = new Set(['description', 'label', 'name'])

/** Changes that appeared, disappeared or were rewired, as against those that only reworded something. */
function structuralOf(d: DiffResponse) {
  const x = d.diff
  return (
    (x.added_nodes ?? []).length + (x.removed_nodes ?? []).length + (x.added_edges ?? []).length + (x.removed_edges ?? []).length +
    (x.added_entries ?? []).length + (x.removed_entries ?? []).length + (x.moved ?? []).length +
    (x.changed ?? []).filter((c) => !wordFields.has(c.field)).length
  )
}
function wordsOf(d: DiffResponse) {
  return (d.diff.changed ?? []).filter((c) => wordFields.has(c.field)).length
}

/**
 * What changed, said as a reader would say it (Surface Screens): the system's own elements one by
 * one, by name; the parts inside a container counted under it; rewording kept apart, with a hollow tag.
 */
function ChangeList({ d, name, kind }: { d: DiffResponse; name: (id: string) => string; kind: (id: string) => string | undefined }) {
  if (d.empty) return <p className="muted small">Nothing changed between v{d.from} and v{d.to}.</p>
  const x = d.diff
  const rows: { g: string; t: string; words?: boolean }[] = []
  const counted = (list: { kind: string; parent?: string; name: string }[], verb: string, g: string) => {
    const byParent = new Map<string, string[]>()
    for (const n of list) {
      if (!part(n.kind)) {
        rows.push({ g, t: `${n.name} ${verb}` })
        continue
      }
      byParent.set(n.parent ?? '', [...(byParent.get(n.parent ?? '') ?? []), n.name])
    }
    for (const [p, names] of byParent) {
      const shown = names.slice(0, 3).join(', ') + (names.length > 3 ? ` and ${names.length - 3} more` : '')
      rows.push({ g, t: names.length === 1 ? `${names[0]} ${verb} in ${name(p)}` : `${names.length} parts ${verb} in ${name(p)} — ${shown}` })
    }
  }
  for (const m of x.moved ?? []) rows.push({ g: '→', t: m.class === 'renamed' ? `${m.was} renamed to ${m.is}` : `${m.is} crossed the system boundary` })
  counted(x.added_nodes ?? [], 'appeared', '+')
  counted(x.removed_nodes ?? [], 'removed', '−')
  const top = (e: { from: string; to: string }) => !part(kind(e.from)) && !part(kind(e.to))
  for (const e of (x.added_edges ?? []).filter(top)) rows.push({ g: '+', t: `${name(e.from)} → ${name(e.to)}` })
  for (const e of (x.removed_edges ?? []).filter(top)) rows.push({ g: '−', t: `${name(e.from)} → ${name(e.to)} removed` })
  const inner = [...(x.added_edges ?? []), ...(x.removed_edges ?? [])].filter((e) => !top(e)).length
  if (inner) rows.push({ g: '±', t: `${inner} ${inner === 1 ? 'use' : 'uses'} between parts rewired` })
  const added = (x.added_entries ?? []).length
  const removed = (x.removed_entries ?? []).length
  if (added || removed) rows.push({ g: '±', t: `${added} ways in appeared, ${removed} disappeared` })
  for (const c of (x.changed ?? []).filter((c) => !c.field.startsWith('column '))) {
    const words = wordFields.has(c.field)
    rows.push({ g: '±', words, t: words ? `${name(c.element)} ${c.field} reworded (words only)` : `${name(c.element)} ${c.field} ${c.before || '—'} → ${c.after || '—'}` })
  }
  const columns = (x.changed ?? []).filter((c) => c.field.startsWith('column ')).length
  if (columns) rows.push({ g: '±', t: `${columns} ${columns === 1 ? 'column' : 'columns'} changed` })
  return (
    <ul className="change-list">
      {rows.slice(0, 6).map((r, i) => (
        <li key={i}>
          <span className={`delta-tag ${r.words ? 'hollow' : ''}`}>{r.g}</span>
          <span>{r.t}</span>
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
