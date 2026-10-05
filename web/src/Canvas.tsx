import { useEffect, useRef, useState } from 'react'
import type { Model, ModelResponse } from './api'
import { useApi } from './api'
import { Failure, Loading } from './ui/marks'
import { Inspector } from './screens/Inspector'

type View = 'container' | 'context'

/**
 * The diagram is the SVG the engine commits to the repository, byte for byte — the app draws
 * nothing itself. Every element in it is a <g id="…">, which is all the page needs to make it
 * clickable.
 */
export function Canvas(props: {
  version: number | null
  level: View
  selected: string | null
  onLevel: (v: View) => void
  onSelect: (id: string | null) => void
}) {
  const { version, level: view, selected, onLevel: setView, onSelect: setSelected } = props
  const [fit, setFit] = useState(true)
  const q = version ? `&version=${version}` : ''
  const svg = useApi<string>(`/api/svg?view=${view}${q}`, 'text')
  const model = useApi<ModelResponse>(`/api/model?${q.slice(1)}`)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    ref.current?.querySelectorAll('g[id]').forEach((g) => g.classList.toggle('selected', g.id === selected))
  }, [selected, svg.data])

  const current: Model | undefined = model.data?.[view]

  return (
    <div className="canvas-layout">
      <section>
        <div className="toolbar">
          <div className="segmented">
            {(['context', 'container'] as const).map((v) => (
              <button key={v} className={view === v ? 'active' : ''} onClick={() => setView(v)}>
                {v === 'context' ? 'System context' : 'Containers'}
              </button>
            ))}
          </div>
          <button onClick={() => setFit(!fit)}>{fit ? 'Actual size' : 'Fit to width'}</button>
          <span className="muted small">
            Click an element to inspect it. <i>Italic</i> text was written by the model.
          </span>
        </div>
        {svg.error && <Failure error={svg.error} />}
        {svg.loading && <Loading />}
        {svg.data && (
          <div
            ref={ref}
            className={`canvas ${fit ? 'fit' : ''}`}
            onClick={(e) => setSelected((e.target as Element).closest('g[id]')?.id ?? null)}
            dangerouslySetInnerHTML={{ __html: svg.data }}
          />
        )}
      </section>
      {model.error && <Failure error={model.error} />}
      {current && <Inspector model={current} id={selected} onSelect={setSelected} />}
    </div>
  )
}
