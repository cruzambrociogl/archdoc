import type { Flow } from '../api'
import { Cite } from './Cite'
import { TruthMark } from './marks'

// A flow as a sequence diagram (F-12): a lifeline per participant, a numbered arrow per step, in
// the order the code makes the calls. Everything drawn is a stored step; the numbers tie each
// arrow to its line in the list beneath, where every step is cited.

const COL = 168
const HEAD = 44
const ROW = 30
const PAD = 16

export function Sequence({ flow, onTable, onJob }: { flow: Flow; onTable?: (element: string) => void; onJob?: (entry: string) => void }) {
  const cols = new Map(flow.participants.map((p, i) => [p.id, i]))
  const x = (id: string) => PAD + (cols.get(id) ?? 0) * COL + COL / 2
  // A step names its participants by ID; a reader wants the class, the module's name, the table.
  const names = new Map(flow.participants.map((p) => [p.id, p.kind === 'table' ? `table ${p.name}` : p.kind === 'job' ? `job ${p.name}` : p.kind === 'unresolved' ? 'a computed address' : p.name]))
  const label = (id: string) => names.get(id) ?? id
  // A table opens in the data view; a queued job opens its own flow.
  const open = (p: Flow['participants'][number]) =>
    p.element && p.kind === 'table' && onTable ? () => onTable(p.element!) : p.element && p.kind === 'job' && onJob ? () => onJob(p.element!) : undefined
  const width = PAD * 2 + flow.participants.length * COL
  const height = HEAD + 16 + flow.steps.length * ROW + 12

  return (
    <div className="sequence">
      <div className="sequence-scroll">
        <svg width={width} height={height} viewBox={`0 0 ${width} ${height}`} role="img" aria-label="Sequence diagram">
          <defs>
            <marker id="seq-arrow" viewBox="0 0 8 8" refX="7" refY="4" markerWidth="7" markerHeight="7" orient="auto-start-reverse">
              <path d="M0 0L8 4L0 8z" fill="var(--edge)" />
            </marker>
          </defs>
          {flow.participants.map((p) => {
            const cx = x(p.id)
            return (
              <g key={p.id} className={`seq-part seq-${p.kind}`}>
                <line x1={cx} x2={cx} y1={HEAD} y2={height - 6} className="seq-life" />
                <rect x={cx - COL / 2 + 8} y={6} width={COL - 16} height={HEAD - 12} rx={3} />
                <text
                  x={cx}
                  y={HEAD / 2 + 4}
                  textAnchor="middle"
                  className={open(p) ? 'seq-link' : undefined}
                  onClick={open(p)}
                >
                  {p.kind === 'unresolved' ? '? computed address' : p.name.length > 22 ? p.name.slice(0, 21) + '…' : p.name}
                </text>
              </g>
            )
          })}
          {flow.steps.map((s, i) => {
            const y = HEAD + 16 + i * ROW
            const x1 = x(s.from)
            const x2 = x(s.to)
            const label = `${i + 1} · ${s.call}`
            if (x1 === x2) {
              // A call on itself: a small loop to the right of the lifeline.
              return (
                <g key={i} className="seq-step">
                  <path d={`M${x1},${y} h22 v12 h-20`} fill="none" markerEnd="url(#seq-arrow)" />
                  <text x={x1 + 28} y={y + 9}>
                    {label}
                  </text>
                </g>
              )
            }
            const left = Math.min(x1, x2)
            return (
              <g key={i} className={`seq-step ${s.to === 'unresolved' ? 'seq-unresolved' : s.to.startsWith('job:') ? 'seq-later' : ''}`}>
                <line x1={x1} x2={x2 + (x2 > x1 ? -3 : 3)} y1={y} y2={y} markerEnd="url(#seq-arrow)" />
                <text x={left + Math.abs(x2 - x1) / 2} y={y - 5} textAnchor="middle">
                  {label}
                </text>
              </g>
            )
          })}
        </svg>
      </div>
      <details className="sequence-steps">
        <summary>
          {flow.steps.length} steps, each at its line{flow.cut ? ' · cut at four calls deep or forty steps' : ''}
        </summary>
        <ol>
          {flow.steps.map((s, i) => (
            <li key={i} style={{ paddingLeft: s.depth * 14 }}>
              <span className="mono small">
                {label(s.from)} → {label(s.to)} · {s.call}
              </span>
              {s.note && (
                <span className="small">
                  {' '}
                  {s.to === 'unresolved' && <TruthMark state="unresolved" />} <span className={s.to === 'unresolved' ? 'mono' : 'muted'}>{s.note}</span>
                </span>
              )}{' '}
              <Cite p={s.provenance} compact />
            </li>
          ))}
        </ol>
      </details>
    </div>
  )
}
