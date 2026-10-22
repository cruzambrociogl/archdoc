import type { Capabilities } from '../source'
import { TruthMark } from './marks'

// The four marks that run through the docs (platform-surface §4.7; App Findings, round 4, problem 2):
// each can be selected, and opens the rail with what it means and what you can do about it.
export type MarkKind = 'proven' | 'ai' | 'unres' | 'notread'

export interface MarkInfo {
  label: string
  /** What it means, in a sentence. */
  means: string
  /** What you can do about it, where you are. */
  how: string
}

export function marks(can: Capabilities): Record<MarkKind, MarkInfo> {
  return {
    proven: {
      label: 'Proven',
      means: 'Read from the code, with the file and line that says so.',
      how: 'Its citation — the file and the line — opens where the code is.',
    },
    ai: {
      label: 'Written by AI',
      means: 'Written by a model, from proven facts only. The facts underneath are proven; the wording is the model’s.',
      how: 'Nothing is written by AI unless someone asked: archdoc label or explain, which say what they will send, and what it costs, before sending.',
    },
    unres: {
      label: 'Couldn’t be resolved',
      means: 'archdoc found evidence that something is there — a call, an address — but couldn’t tell what, because it is only known when the app runs. It is shown where it happens, and never guessed.',
      how:
        can.place === 'published'
          ? 'The repository’s owner can name it with a correction.'
          : 'If you know what it is, name it with a correction in .archdoc/rules.yaml; Settings lists the corrections in force.',
    },
    notread: {
      label: 'Not read',
      means: 'Files in a language archdoc doesn’t parse yet. Nothing in them is shown, and nothing about them is guessed.',
      how: 'About these docs lists what was not read.',
    },
  }
}

export const markOrder: MarkKind[] = ['proven', 'ai', 'unres', 'notread']

/** A mark's glyph, as the docs draw it. */
export function MarkGlyph({ kind }: { kind: MarkKind }) {
  if (kind === 'notread') return <span className="mark mark-notread" aria-hidden="true" />
  return <TruthMark state={kind === 'proven' ? 'proven' : kind === 'ai' ? 'interpreted' : 'unresolved'} />
}
