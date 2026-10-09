import type { ReactNode } from 'react'
import { published } from '../api'
import { plan } from '../planned'
import type { Route } from '../route'

// A placeholder for something the spec designs and the app does not do yet: dashed, labelled
// "planned", and a link to the page that says what it will be. Shown only in the local app — a
// published site is for readers of the documentation, not of archdoc's plans.

type Go = (r: Partial<Route>) => void

/** A small dashed chip, where a button or a control will go. */
export function Planned({ id, go, children }: { id: string; go: Go; children?: ReactNode }) {
  const p = plan(id)
  if (published || !p) return null
  return (
    <button className="planned" title={`Planned — ${p.what} (surface-spec ${p.spec})`} onClick={() => go({ screen: 'planned', focus: id })}>
      {children ?? p.title}
      <span className="planned-tag">planned</span>
    </button>
  )
}

/** A dashed slot, where a whole panel will go: its title and what it will do. */
export function PlannedSlot({ id, go }: { id: string; go: Go }) {
  const p = plan(id)
  if (published || !p) return null
  return (
    <button className="planned-slot" onClick={() => go({ screen: 'planned', focus: id })}>
      <span className="planned-slot-title">
        {p.title} <span className="planned-tag">planned</span>
      </span>
      <span className="planned-slot-what">{p.what}</span>
    </button>
  )
}
