import { useMemo } from 'react'
import type { Entry, Flow, ModelResponse, Node } from '../api'
import { componentLevel, dataLevel, useApi } from '../api'
import type { Route } from '../route'
import { Cite } from '../ui/Cite'
import { Eyebrow, Failure, Loading, TruthMark } from '../ui/marks'
import { Sequence } from '../ui/Sequence'

/**
 * What the system does (F-13), read from where its code says so: every route a container serves,
 * grouped by the class that handles it, each with its line. A description is shown only when the
 * code states one — a summary in its own decorators — so nothing here is interpretation. Calls the
 * code makes to an address computed at run time close the page: seen, and not tied to anything.
 */
export function Features(props: { version: number | null; route: Route; go: (r: Partial<Route>, o?: { replace?: boolean; keep?: boolean }) => void }) {
  const q = props.version ? `?version=${props.version}` : ''
  const m = useApi<ModelResponse>(`/api/model${q}`)
  const find = (props.route.q ?? '').trim().toLowerCase()
  const open = props.route.focus

  const groups = useMemo(() => {
    const entries = m.data?.model.entries ?? []
    const hit = (e: Entry) =>
      !find || [e.method, e.path, e.handler, e.summary ?? ''].some((s) => s.toLowerCase().includes(find))
    const byContainer = new Map<string, Map<string, Entry[]>>()
    for (const e of entries.filter(hit)) {
      const cls = e.handler.split('.')[0]
      const c = byContainer.get(e.container) ?? new Map<string, Entry[]>()
      c.set(cls, [...(c.get(cls) ?? []), e])
      byContainer.set(e.container, c)
    }
    return [...byContainer.entries()].map(([container, classes]) => ({
      container,
      classes: [...classes.entries()].sort(([a], [b]) => a.localeCompare(b)),
    }))
  }, [m.data, find])

  if (m.error) return <Failure error={m.error} />
  if (!m.data) return <Loading />
  const model = m.data.model
  const entries = model.entries ?? []
  const unresolved = model.unresolved ?? []
  const nodes = new Map((model.nodes ?? []).map((n) => [n.id, n]))
  const flows = new Map((model.flows ?? []).map((f) => [f.entry, f]))
  const name = (id?: string) => (id ? (nodes.get(id)?.name ?? id) : '')
  const described = entries.filter((e) => e.summary).length
  const containers = new Set(entries.map((e) => e.container)).size
  const shown = groups.reduce((n, g) => n + g.classes.reduce((k, [, es]) => k + es.length, 0), 0)
  const setFind = (v: string) => props.go({ screen: 'features', q: v || undefined, focus: open }, { replace: true })
  const toggle = (id: string) => props.go({ screen: 'features', q: props.route.q, focus: open === id ? undefined : id }, { replace: true })

  return (
    <div className="reading wide">
      <Eyebrow>Features · what the system does</Eyebrow>
      <h1 className="page-title">The ways in</h1>
      {entries.length === 0 ? (
        <p className="lede">
          No routes were found in the code archdoc reads. Routes are read from NestJS controllers today; other frameworks follow.
        </p>
      ) : (
        <p className="lede">
          {entries.length} routes, read from the code of {containers === 1 ? name(entries[0].container) : `${containers} containers`}. {described} of them are described by
          the code itself — a summary its decorators state — and the rest show only their handler: nothing here is written by a model.
        </p>
      )}

      {entries.length > 0 && (
        <label className="features-find">
          <span>/</span>
          <input value={props.route.q ?? ''} placeholder="Filter by path, handler or description" onChange={(e) => setFind(e.target.value)} />
          {find && <span className="mono small muted">{shown} shown</span>}
        </label>
      )}

      {groups.map((g) => (
        <section key={g.container} className="features-container">
          {containers > 1 && <h2 className="section-title">{name(g.container)}</h2>}
          {g.classes.map(([cls, es]) => (
            <div key={cls} className="feature-group">
              <div className="feature-head">
                <span className="strong">{cls.replace(/Controller$/, '') || cls}</span>
                <span className="mono small muted">
                  {es.length} {es.length === 1 ? 'route' : 'routes'} · {cls}
                </span>
                {es[0].component && (
                  <button
                    className="text-link small"
                    onClick={() => props.go({ screen: 'explorer', level: componentLevel(g.container), focus: es[0].component })}
                    title="Open the component that holds this handler"
                  >
                    in {name(es[0].component) || componentName(es[0].component)} ⤴
                  </button>
                )}
              </div>
              {es.map((e) => (
                <FeatureRow
                  key={e.id}
                  e={e}
                  open={open === e.id}
                  onToggle={() => toggle(e.id)}
                  nodes={nodes}
                  flow={flows.get(e.id)}
                  onTable={(id) => props.go({ screen: 'explorer', level: dataLevel(g.container), focus: id })}
                />
              ))}
            </div>
          ))}
        </section>
      ))}

      {unresolved.length > 0 && (
        <section className="features-unresolved">
          <Eyebrow>Calls whose target is computed at run time · {unresolved.length}</Eyebrow>
          <p className="lede-quiet">The code makes these calls; nothing in it names where they go. They are listed, not drawn, and no arrow is guessed.</p>
          {unresolved.map((u, i) => (
            <div key={i} className="gap-row static">
              <span>
                <TruthMark state="unresolved" /> <span className="mono">{u.what}</span>
                <span className="muted small"> · {name(u.container)}</span>
              </span>
              <Cite p={u.provenance} compact />
            </div>
          ))}
        </section>
      )}
    </div>
  )
}

function FeatureRow({
  e,
  open,
  onToggle,
  nodes,
  flow,
  onTable,
}: {
  e: Entry
  open: boolean
  onToggle: () => void
  nodes: Map<string, Node>
  flow?: Flow
  onTable: (id: string) => void
}) {
  return (
    <div className={`feature ${open ? 'open' : ''}`}>
      <button className="feature-row" onClick={onToggle} aria-expanded={open}>
        <span className="method">{e.method}</span>
        <span className="mono feature-path">{e.path}</span>
        <span className={`feature-summary ${e.summary ? '' : 'muted'}`}>{e.summary ?? e.handler}</span>
      </button>
      {open && (
        <div className="feature-detail">
          <Fact k="declared at">
            <Cite p={e.provenance} />
          </Fact>
          <Fact k="handler">
            <span className="mono">{e.handler}</span>
          </Fact>
          {e.summary && (
            <Fact k="described">
              {e.summary} <Cite p={e.summary_provenance} compact />
            </Fact>
          )}
          {e.path_note && <Fact k="path">{e.path_note}</Fact>}
          {e.prefix_provenance?.file && (
            <Fact k="prefix">
              the global prefix, set at <Cite p={e.prefix_provenance} compact />
            </Fact>
          )}
          {(e.uses ?? []).length > 0 && (
            <Fact k="given">
              <span className="feature-uses">
                {e.uses!.map((u) => (
                  <span key={u.name} className="feature-use">
                    {u.how === 'unresolved' && <TruthMark state="unresolved" />}
                    <span className="mono">{u.name}</span>
                    {u.how === 'name' ? (
                      <>
                        <span className="muted small"> · {nodes.get(u.component ?? '')?.name ?? 'declared'}, by name</span> <Cite p={u.provenance} compact />
                      </>
                    ) : (
                      <span className="muted small"> · not declared in this code</span>
                    )}
                  </span>
                ))}
              </span>
            </Fact>
          )}
          {flow && (
            <div className="feature-flow">
              <Eyebrow>What it sets off</Eyebrow>
              <Sequence flow={flow} onTable={onTable} />
            </div>
          )}
        </div>
      )}
    </div>
  )
}

function Fact({ k, children }: { k: string; children: React.ReactNode }) {
  return (
    <div className="fact-row">
      <span className="fact-key">{k}</span>
      <span className="fact-value">{children}</span>
    </div>
  )
}

const componentName = (id: string) => id.slice(id.indexOf('/') + 1)
