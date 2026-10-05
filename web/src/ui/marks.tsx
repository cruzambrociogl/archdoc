import type { ReactNode } from 'react'

// The three truth states (surface-spec §0, §6.1). Each has its own form, so they survive greyscale,
// print and colour blindness: proven is a solid square, interpreted a slate diamond, unresolved a
// dotted ochre circle with a question mark. Colour only reinforces.

export type Truth = 'proven' | 'interpreted' | 'unresolved'

export function TruthMark({ state }: { state: Truth }) {
  if (state === 'unresolved') return <span className="mark mark-unresolved" aria-label="unresolved">?</span>
  return <span className={`mark mark-${state}`} aria-label={state} />
}

const labels: Record<Truth, string> = { proven: 'Proven', interpreted: 'Interpreted', unresolved: 'Unresolved' }

/** A labelled state, as the inspector header and the legend show it. */
export function TruthChip({ state }: { state: Truth }) {
  return (
    <span className={`truth-chip truth-${state}`}>
      <TruthMark state={state} />
      {labels[state]}
    </span>
  )
}

export function Loading() {
  return <p className="state-line">Loading…</p>
}

export function Failure({ error }: { error: string }) {
  return <p className="state-line state-error">{error}</p>
}

/** An eyebrow: the small uppercase label that heads a section. */
export function Eyebrow({ children }: { children: ReactNode }) {
  return <div className="eyebrow">{children}</div>
}
