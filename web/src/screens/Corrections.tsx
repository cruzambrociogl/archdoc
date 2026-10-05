import type { Op, RulesResponse } from '../api'
import { useApi, useEditorLink } from '../api'
import { Eyebrow, Failure, Loading } from '../ui/marks'

type Rule = RulesResponse['rules'][number]

/** A rule as it reads in rules.yaml, rebuilt from what was parsed. */
function asYaml(r: Rule): string {
  const inline = (o: Record<string, string>) =>
    `{ ${Object.entries(o)
      .filter(([, v]) => v)
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([k, v]) => `${k}: ${v}`)
      .join(', ')} }`
  const lines = [r.edge ? `- edge: { from: ${r.edge.from}, to: ${r.edge.to} }` : `- match: ${inline(r.match as unknown as Record<string, string>)}`]
  if (r.set && Object.keys(r.set).length) lines.push(`  set: ${inline(r.set)}`)
  if (r.exclude) lines.push('  exclude: true')
  if (r.remove) lines.push('  remove: true')
  return lines.join('\n')
}

const opText = (o: Op) => `${o.kind.replace(/_/g, ' ')}${o.to ? ` → ${o.to}` : ''}${o.value ? ` = ${o.value}` : ''}`

/**
 * What a person corrected (surface-spec §5.13). Rules are compiled against the model as extracted
 * and applied after the model, so a person's correction always wins — and survives every
 * regeneration. Each rule shows what it matched and what it changed; one that matched nothing is
 * a finding, not a silent no-op.
 */
export function Corrections() {
  const r = useApi<RulesResponse>('/api/rules')
  const link = useEditorLink()
  if (r.error) return <Failure error={r.error} />
  if (!r.data) return <Loading />
  const { file, rules, findings } = r.data
  const ops = r.data.operations ?? []

  if (!r.data.exists) {
    return (
      <div className="reading">
        <Eyebrow>Corrections · rules.yaml</Eyebrow>
        <h1 className="page-title small">Nothing has been corrected</h1>
        <p className="lede">
          This repository has no <span className="mono">rules.yaml</span>. A rule is a correction that survives every
          regeneration: rename an element, reclassify it, describe it, exclude it, or add a relationship the configuration
          cannot show.
        </p>
        <pre className="code-block">{`- match: { name: supavisor }
  set: { kind: proxy, description: Pools Postgres connections }
- match: { image: "*/vector*" }
  exclude: true`}</pre>
      </div>
    )
  }

  return (
    <div className="reading wide">
      <Eyebrow>
        Corrections · {link(file) ? <a href={link(file)}>{file}</a> : file} · {rules.length} {rules.length === 1 ? 'rule' : 'rules'}
      </Eyebrow>
      <div className="page-title-row">
        <h1 className="page-title small">What a person corrected</h1>
        <p className="title-note">Rules are applied after the model, so a person's correction always wins, and they survive every regeneration.</p>
      </div>

      {!!findings?.length && (
        <>
          <Eyebrow>Findings · {findings.length}</Eyebrow>
          <div className="findings-grid">
            {findings.map((f, i) => {
              const at = f.element.match(/:(\d+)$/)
              return (
                <div key={i} className="finding">
                  <div className="finding-head">
                    <span className={`owner-chip ${f.severity === 'error' ? '' : 'dashed-chip'}`}>{f.severity}</span>
                    <span className="mono small muted">{f.rule}</span>
                    {at ? (
                      <a className="cite" href={link(file, Number(at[1]))}>
                        <span className="glyph glyph-rules" />
                        <span className="cite-ref">
                          <bdi>
                            {file}
                            <span className="cite-line">:{at[1]}</span>
                          </bdi>
                        </span>
                      </a>
                    ) : (
                      <span className="mono small">{f.element}</span>
                    )}
                  </div>
                  <p className="finding-text">{f.message}</p>
                </div>
              )
            })}
          </div>
        </>
      )}

      <div className="rules-table">
        <div className="rules-row rules-head">
          <span>Rule</span>
          <span>Matched</span>
          <span>Changed</span>
          <span>State</span>
        </div>
        {rules.map((rule) => {
          const did = ops.filter((o) => o.provenance?.line === rule.line)
          const targets = [...new Set(did.map((o) => o.target))].sort()
          return (
            <div key={rule.line} className="rules-row">
              <div>
                <a className="cite" href={link(file, rule.line)} title="Asserted by a person in rules.yaml — open at the line">
                  <span className="glyph glyph-rules" />
                  <span className="cite-ref">
                    <bdi>
                      {file}
                      <span className="cite-line">:{rule.line}</span>
                    </bdi>
                  </span>
                </a>
                <pre className="rule-yaml">{asYaml(rule)}</pre>
              </div>
              <div className="mono small">{targets.length ? targets.join(', ') : <span className="muted">nothing</span>}</div>
              <div className="small">
                {did.length ? did.map((o, i) => <div key={i}>{opText(o)}</div>) : <span className="muted">—</span>}
              </div>
              <div>
                <span className={`owner-chip ${did.length ? '' : 'dashed-chip'}`}>{did.length ? 'applied' : 'matched nothing'}</span>
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
