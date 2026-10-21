import { useEffect } from 'react'
import { planned } from '../planned'
import type { Route } from '../route'
import { Eyebrow } from '../ui/marks'

/**
 * Not built yet: everything the surface spec designs that the app does not do, in one place, each
 * with what it will do and where it is specified. The placeholders around the app open here.
 */
export function Planned(props: { route: Route }) {
  const areas = [...new Set(planned.map((p) => p.area))]
  useEffect(() => {
    if (props.route.focus) document.getElementById(`plan-${props.route.focus}`)?.scrollIntoView({ block: 'center' })
  }, [props.route.focus])
  return (
    <div className="reading">
      <Eyebrow>Not built yet · {planned.length} pieces</Eyebrow>
      <h1 className="page-title">Designed, and still to build</h1>
      <p className="lede">
        What the surface design — the Claude Design files, read on 9 Oct — and its spec describe that this app does not do yet. Most have a dashed
        placeholder where they will go; this list is what they point to, so nothing designed is forgotten. The list is{' '}
        <span className="mono">web/src/planned.ts</span>.
      </p>
      {areas.map((a) => (
        <section key={a} className="planned-area">
          <h2 className="section-title">{a}</h2>
          {planned
            .filter((p) => p.area === a)
            .map((p) => (
              <div key={p.id} id={`plan-${p.id}`} className={`planned-row ${props.route.focus === p.id ? 'focused' : ''}`}>
                <div className="planned-row-head">
                  <span className="strong">{p.title}</span>
                  <span className="mono small muted">
                    surface-spec {p.spec}
                    {p.design ? ` · design: ${p.design}` : ''}
                    {p.waits ? ` · ${p.waits}` : ''}
                    {p.listOnly ? ' · no placeholder on screen' : ''}
                  </span>
                </div>
                <p className="planned-row-what">{p.what}</p>
              </div>
            ))}
        </section>
      ))}
    </div>
  )
}
