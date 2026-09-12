import type { CompletenessResponse } from './api'
import { editorLink, useApi, when } from './api'
import { Failure, Loading } from './Prov'

const explain: Record<string, string> = {
  missing: 'The file is gone. The next generate recreates the stub.',
  'not started': 'Still the stub archdoc wrote.',
  written: 'Someone has written in it since the stub.',
  'may be stale': 'Written before the architecture last changed — it may describe a system that no longer exists.',
}

/**
 * How far the human sections have come. Measured from the filesystem alone — size and
 * modification time against the stub archdoc recorded — so no one's writing is ever read.
 */
export function Completeness({ version }: { version: number | null }) {
  const c = useApi<CompletenessResponse>(`/api/completeness${version ? `?version=${version}` : ''}`)
  if (c.error) return <Failure error={c.error} />
  if (!c.data) return <Loading />
  const { sections, dir } = c.data
  const done = sections.filter((s) => s.state === 'written').length

  return (
    <>
      <div className="toolbar">
        <h2>Completeness</h2>
        <span>
          {done} of {sections.length} sections written
        </span>
        <span className="muted small">architecture last changed {when(c.data.architecture_changed)}</span>
      </div>
      <div className="meter">
        <div style={{ width: `${(done / Math.max(sections.length, 1)) * 100}%` }} />
      </div>
      <table>
        <thead>
          <tr>
            <th>Section</th>
            <th>State</th>
            <th>Last modified</th>
          </tr>
        </thead>
        <tbody>
          {sections.map((s) => (
            <tr key={s.file}>
              <td>
                {s.state === 'missing' ? (
                  <>
                    {s.number}. {s.title}
                  </>
                ) : (
                  <a href={editorLink(dir, s.file)}>
                    {s.number}. {s.title}
                  </a>
                )}
              </td>
              <td>
                <span className={`state ${s.state.replace(/ /g, '-')}`} title={explain[s.state]}>
                  {s.state}
                </span>
              </td>
              <td className="muted small">{when(s.modified)}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <p className="muted small">
        archdoc never opens these files: it compares their size and modification time with the stub it wrote.
      </p>
    </>
  )
}
