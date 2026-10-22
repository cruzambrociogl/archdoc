import { createContext, useContext, useEffect } from 'react'
import { useCapabilities } from '../source'
import type { Route } from '../route'
import type { MarkKind } from '../ui/markInfo'
import { MarkGlyph, markOrder, marks } from '../ui/markInfo'
import { Impact } from '../screens/Impact'
import { Ask } from '../screens/Ask'

/** What the rail shows: a mark, or what an element touches. */
export type RailContent = { kind: 'mark'; mark: MarkKind } | { kind: 'impact'; element: string } | { kind: 'ask'; question: string }

/** Opens the rail from anywhere in the app: an inspector, a component page, a search result. */
const RailContext = createContext<(c: RailContent) => void>(() => {})
export const RailProvider = RailContext.Provider
export const useRail = () => useContext(RailContext)

const titles: Record<RailContent['kind'], string> = { mark: 'What this mark means', impact: 'What does this touch?', ask: 'Ask' }

/**
 * The right rail (platform-surface §4.3): opened when something is selected, closed otherwise. Beside
 * the page on a wide screen; on a phone, a sheet from the bottom over a scrim (App Findings, round 4).
 * Esc closes it.
 */
export function Rail(props: { content: RailContent; onOpen: (c: RailContent) => void; onClose: () => void; version: number | null; go: (r: Partial<Route>) => void; onSearch: () => void }) {
  const can = useCapabilities()
  const { onClose } = props
  useEffect(() => {
    const esc = (e: KeyboardEvent) => e.key === 'Escape' && onClose()
    document.addEventListener('keydown', esc)
    return () => document.removeEventListener('keydown', esc)
  }, [onClose])

  const c = props.content
  return (
    <>
      <div className="side-rail-scrim" onClick={onClose} />
      <aside className="side-rail" aria-label={titles[c.kind]}>
        <div className="side-rail-head">
          <span className="pop-q">{titles[c.kind]}</span>
          <button className="side-rail-close" onClick={onClose} aria-label="Close">
            ✕
          </button>
        </div>
        <div className="side-rail-body">
          {c.kind === 'mark' ? (
            <MarkPanel mark={c.mark} onOpen={props.onOpen} can={can} />
          ) : c.kind === 'ask' ? (
            <Ask key={c.question} question={c.question} go={props.go} onFollowUp={props.onSearch} />
          ) : (
            <Impact element={c.element} version={props.version} go={props.go} />
          )}
        </div>
      </aside>
    </>
  )
}

function MarkPanel(props: { mark: MarkKind; onOpen: (c: RailContent) => void; can: ReturnType<typeof useCapabilities> }) {
  const all = marks(props.can)
  const m = all[props.mark]
  return (
    <>
      <div className={`side-rail-mark is-${props.mark}`}>
        <MarkGlyph kind={props.mark} />
        <span className="side-rail-mark-label">{m.label}</span>
      </div>
      <p className="side-rail-means">{m.means}</p>
      <p className="side-rail-how">{m.how}</p>
      <div className="pop-q side-rail-every">Every mark</div>
      {markOrder.map((k) => (
        <button key={k} className={`side-rail-row ${k === props.mark ? 'on' : ''}`} onClick={() => props.onOpen({ kind: 'mark', mark: k })}>
          <MarkGlyph kind={k} />
          <span className="side-rail-row-label">{all[k].label}</span>
        </button>
      ))}
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
