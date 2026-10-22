import { useEffect } from 'react'
import { useCapabilities } from '../source'
import type { MarkKind } from '../ui/markInfo'
import { MarkGlyph, markOrder, marks } from '../ui/markInfo'

export type RailContent = { kind: 'mark'; mark: MarkKind }

/**
 * The right rail (platform-surface §4.3): opened when something is selected, closed otherwise. Beside
 * the page on a wide screen; on a phone, a sheet from the bottom over a scrim (App Findings, round 4).
 * Esc closes it.
 */
export function Rail(props: { content: RailContent; onOpen: (c: RailContent) => void; onClose: () => void }) {
  const can = useCapabilities()
  const { onClose } = props
  useEffect(() => {
    const esc = (e: KeyboardEvent) => e.key === 'Escape' && onClose()
    document.addEventListener('keydown', esc)
    return () => document.removeEventListener('keydown', esc)
  }, [onClose])

  const all = marks(can)
  const m = all[props.content.mark]
  return (
    <>
      <div className="side-rail-scrim" onClick={onClose} />
      <aside className="side-rail" aria-label="What this mark means">
        <div className="side-rail-head">
          <span className="pop-q">What this mark means</span>
          <button className="side-rail-close" onClick={onClose} aria-label="Close">
            ✕
          </button>
        </div>
        <div className="side-rail-body">
          <div className={`side-rail-mark is-${props.content.mark}`}>
            <MarkGlyph kind={props.content.mark} />
            <span className="side-rail-mark-label">{m.label}</span>
          </div>
          <p className="side-rail-means">{m.means}</p>
          <p className="side-rail-how">{m.how}</p>
          <div className="pop-q side-rail-every">Every mark</div>
          {markOrder.map((k) => (
            <button key={k} className={`side-rail-row ${k === props.content.mark ? 'on' : ''}`} onClick={() => props.onOpen({ kind: 'mark', mark: k })}>
              <MarkGlyph kind={k} />
              <span className="side-rail-row-label">{all[k].label}</span>
            </button>
          ))}
        </div>
      </aside>
    </>
  )
}

const seenKey = 'archdoc.marks-seen'

/** Whether this browser has dismissed the first-visit band. Storage may be blocked; then it shows. */
export function marksSeen(): boolean {
  try {
    return localStorage.getItem(seenKey) === '1'
  } catch {
    return false
  }
}

/**
 * The first-visit band (App Findings, round 4, problem 2): one line under the top bar naming the
 * three marks, each of which opens the rail. "Got it" is remembered on this browser.
 */
export function MarksBand(props: { onOpen: (c: RailContent) => void; onDismiss: () => void }) {
  const all = marks(useCapabilities())
  return (
    <div className="marks-band" role="note">
      <span>Three marks run through these docs. Select any of them to see what it means.</span>
      <span className="marks-band-marks">
        {(['proven', 'ai', 'unres'] as MarkKind[]).map((k) => (
          <button key={k} className={`marks-band-mark is-${k}`} onClick={() => props.onOpen({ kind: 'mark', mark: k })}>
            <MarkGlyph kind={k} />
            {all[k].label}
          </button>
        ))}
      </span>
      <button
        className="tool marks-band-done"
        onClick={() => {
          try {
            localStorage.setItem(seenKey, '1')
          } catch {
            /* storage blocked: it shows again next time */
          }
          props.onDismiss()
        }}
      >
        Got it
      </button>
    </div>
  )
}
