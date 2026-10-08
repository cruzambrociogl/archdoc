import type { ModelResponse, Package } from '../api'
import { componentLevel, useApi } from '../api'
import type { Route } from '../route'
import { Cite } from '../ui/Cite'
import { Eyebrow, Failure, Loading, TruthMark } from '../ui/marks'

/**
 * What the system relies on (F-14): the packages each container's manifest declares — each at its
 * line — and where the code uses them: how many imports name the package, from which components.
 * A package declared and never imported is listed as such; nothing here is interpretation.
 */
export function Dependencies(props: { version: number | null; route: Route; go: (r: Partial<Route>, o?: { replace?: boolean }) => void }) {
  const q = props.version ? `?version=${props.version}` : ''
  const m = useApi<ModelResponse>(`/api/model${q}`)
  if (m.error) return <Failure error={m.error} />
  if (!m.data) return <Loading />
  const model = m.data.model
  const deps = model.dependencies ?? []
  const names = new Map((model.nodes ?? []).map((n) => [n.id, n.name]))
  const find = (props.route.q ?? '').trim().toLowerCase()
  const byContainer = new Map<string, Package[]>()
  for (const d of deps) {
    if (find && !d.name.toLowerCase().includes(find)) continue
    byContainer.set(d.container, [...(byContainer.get(d.container) ?? []), d])
  }

  return (
    <div className="reading wide">
      <Eyebrow>Dependencies · what it relies on</Eyebrow>
      <h1 className="page-title">Other people's code</h1>
      <p className="lede">
        {deps.length} packages, declared by {new Set(deps.map((d) => d.container)).size} manifests. Each is cited at the line that declares it, with how often
        the code imports it and from which components; build-only tools that nothing imports are left out.
      </p>
      <label className="features-find">
        <span>/</span>
        <input value={props.route.q ?? ''} placeholder="Filter by package name" onChange={(e) => props.go({ screen: 'deps', q: e.target.value || undefined }, { replace: true })} />
      </label>
      {[...byContainer.entries()].map(([container, list]) => (
        <section key={container} className="feature-group">
          <div className="feature-head">
            <span className="strong">{names.get(container) ?? container}</span>
            <span className="mono small muted">{list.length} packages</span>
          </div>
          {[...list]
            .sort((a, b) => b.imports - a.imports || a.name.localeCompare(b.name))
            .map((d) => (
              <div key={d.name} className="feature dep-row">
                <span className="mono">
                  {d.name} <span className="muted small">{d.version}</span>
                  {d.dev && <span className="muted small"> · dev</span>}
                </span>
                <span className="small">
                  {d.imports > 0 ? (
                    <>
                      {d.imports} {d.imports === 1 ? 'import' : 'imports'}
                      {(d.components ?? []).length > 0 && ' · '}
                      {(d.components ?? []).map((c, i) => (
                        <span key={c}>
                          {i > 0 && ', '}
                          <button className="text-link" onClick={() => props.go({ screen: 'explorer', level: componentLevel(container), focus: c })}>
                            {names.get(c) ?? c}
                          </button>
                        </span>
                      ))}
                    </>
                  ) : (
                    <span className="muted">
                      <TruthMark state="unresolved" /> declared, and no import of it found
                    </span>
                  )}
                </span>
                <Cite p={d.provenance} compact />
              </div>
            ))}
        </section>
      ))}
    </div>
  )
}
