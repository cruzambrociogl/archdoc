import type { RulesResponse } from './api'
import { useApi, useEditorLink } from './api'
import { Failure, Loading } from './Prov'

/** What rules.yaml says, and what each rule did to the model as extracted. */
export function Rules() {
  const r = useApi<RulesResponse>('/api/rules')
  const link = useEditorLink()
  if (r.error) return <Failure error={r.error} />
  if (!r.data) return <Loading />
  const { file, rules, findings } = r.data
  const ops = r.data.operations ?? []

  if (!r.data.exists) {
    return (
      <>
        <h2>No rules</h2>
        <p>
          This repository has no <code>rules.yaml</code>. Rules are corrections that survive regeneration: rename an
          element, reclassify it, exclude it, or add a relationship the configuration cannot show.
        </p>
        <pre>{`rules:
  - match: { name: supavisor }
    set: { kind: proxy, description: Pools Postgres connections }
  - match: { image: "*/vector*" }
    exclude: true`}</pre>
      </>
    )
  }

  return (
    <>
      <div className="toolbar">
        <h2>Rules</h2>
        <a href={link(file)}>{file}</a>
        <span className="muted small">
          {rules.length} rules, {ops.length} changes to the model
        </span>
      </div>

      {!!findings?.length && (
        <ul className="plain findings">
          {findings.map((f, i) => (
            <li key={i} className={f.severity}>
              <b>{f.severity}</b> rule at line {f.rule}
              {f.element && <> ({f.element})</>}: {f.message}
            </li>
          ))}
        </ul>
      )}

      <table>
        <thead>
          <tr>
            <th>Line</th>
            <th>Matches</th>
            <th>Says</th>
            <th>Did</th>
          </tr>
        </thead>
        <tbody>
          {rules.map((rule) => {
            const did = ops.filter((o) => o.provenance?.line === rule.line)
            const match = Object.entries(rule.match).filter(([, v]) => v)
            return (
              <tr key={rule.line}>
                <td>
                  <a href={link(file, rule.line)}>{rule.line}</a>
                </td>
                <td className="small">
                  {rule.edge ? (
                    <>
                      relationship {rule.edge.from} → {rule.edge.to}
                    </>
                  ) : (
                    match.map(([k, v]) => (
                      <div key={k}>
                        {k}: <code>{v}</code>
                      </div>
                    ))
                  )}
                </td>
                <td className="small">
                  {rule.exclude && <div>exclude</div>}
                  {rule.remove && <div>remove</div>}
                  {Object.entries(rule.set ?? {})
                    .sort()
                    .map(([k, v]) => (
                      <div key={k}>
                        {k} = {v}
                      </div>
                    ))}
                  {rule.edge && !rule.remove && <div>add relationship</div>}
                </td>
                <td className="small">
                  {did.length ? (
                    did.map((o, i) => (
                      <div key={i}>
                        <span className="mono">{o.kind}</span> {o.target}
                        {o.to && <> → {o.to}</>}
                        {o.value && <> = {o.value}</>}
                      </div>
                    ))
                  ) : (
                    <span className="warn-text">matched nothing</span>
                  )}
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </>
  )
}
