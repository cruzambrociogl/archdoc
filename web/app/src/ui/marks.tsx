import type { ReactNode } from 'react'
import { retryFailed } from '../connection'

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

/**
 * Waiting, and for what (rule 11): never a spinner alone. A page passes its question, which the
 * route already knows, so the page says what it answers before its answer arrives (§4.8).
 */
export function Loading({ what, q }: { what?: string; q?: string }) {
  return (
    <>
      {q && <div className="page-q">{q}</div>}
      <p className="state-line" role="status">
        {what ? `Loading ${what}…` : 'Loading…'}
      </p>
    </>
  )
}

/**
 * A read that failed (platform-surface §4.8): what happened, what is still true, and what to do.
 * `whole` when nothing else can be shown without it.
 */
export function Failure({ error, whole }: { error: string; whole?: boolean }) {
  return (
    <div className="failure" role="alert">
      <p className="state-line state-error">This couldn’t be loaded: {error.replace(/\.$/, '')}.</p>
      <p className="state-line">
        {whole ? 'Nothing can be shown without it.' : 'The rest of the app still works.'}{' '}
        <button className="link-btn" onClick={retryFailed}>
          Try again
        </button>
      </p>
    </div>
  )
}

/** An eyebrow: the small uppercase label that heads a section. */
export function Eyebrow({ children }: { children: ReactNode }) {
  return <div className="eyebrow">{children}</div>
}
