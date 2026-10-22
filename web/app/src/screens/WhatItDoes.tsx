import { useMemo, useState } from 'react'
import type { ModelResponse } from '../api'
import { useApi } from '../api'
import type { Route } from '../route'
import { Cite } from '../ui/Cite'
import { Failure, Loading } from '../ui/marks'
import { featuresOf } from '../features'

/**
 * What it does (platform-surface §4.3, §6.6): what can it do, and how does each thing work? Each
 * feature in one sentence, written by a template from the routes it comes from — proven, never a
 * count — opening onto those routes and, where the engine followed one, what it sets off. The
 * largest five first; the rest one deliberate step away (rule 15). Every entry point, as a list, is
 * in Reference.
 */
export function WhatItDoes(props: { version: number | null; go: (r: Partial<Route>) => void }) {
  const q = props.version ? `?version=${props.version}` : ''
  const m = useApi<ModelResponse>(`/api/model${q}`)
  const [all, setAll] = useState(false)
  const [open, setOpen] = useState<string | null>(null)
  const features = useMemo(() => featuresOf(m.data?.model.entries ?? []), [m.data])

  if (m.error) return <Failure error={m.error} />
  if (!m.data) return <Loading q="What can it do, and how does each thing work?" what="its routes" />
  const flows = new Set((m.data.model.flows ?? []).map((f) => f.entry))
  const shown = all ? features : features.slice(0, 5)

  return (
    <div className="reading what-it-does">
      <div className="page-q">What can it do, and how does each thing work?</div>
      {features.length === 0 ? (
        <p className="story-answer">archdoc found no routes, pages, jobs or commands in the code it read, so there is nothing to say here that would not be a guess.</p>
      ) : (
        <>
          <p className="overview-source">Each sentence is written from the routes beside it — proven, not interpreted. Select one to see them, and what each sets off.</p>
          <ul className="features">
            {shown.map((f) => (
              <li key={f.key} className={`feature ${open === f.key ? 'open' : ''}`}>
                <button className="feature-head" aria-expanded={open === f.key} onClick={() => setOpen(open === f.key ? null : f.key)}>
                  <span className="feature-sentence">{f.sentence}</span>
                  <span className="feature-count">
                    {f.entries.length} {f.entries.length === 1 ? 'way in' : 'ways in'}
                  </span>
                </button>
                {open === f.key && (
                  <ul className="feature-entries">
                    {f.entries.slice(0, 12).map((e) => (
                      <li key={e.id}>
                        <span className="mono">
                          {['page', 'job', 'command'].includes(e.kind) ? e.kind : e.method} {e.path}
                        </span>
                        {e.summary && <span className="muted"> — {e.summary}</span>}
                        <span className="feature-entry-right">
                          {flows.has(e.id) && (
                            <button className="link-btn" onClick={() => props.go({ screen: 'entries', focus: e.id })}>
                              How it works →
                            </button>
                          )}
                          <Cite p={e.provenance} compact />
                        </span>
                      </li>
                    ))}
                    {f.entries.length > 12 && (
                      <li>
                        <button className="link-btn" onClick={() => props.go({ screen: 'entries', q: f.key.startsWith('area:') ? f.key.slice(5) : undefined })}>
                          All {f.entries.length}, in Every entry point →
                        </button>
                      </li>
                    )}
                  </ul>
                )}
              </li>
            ))}
          </ul>
          {features.length > 5 && (
            <button className="link-btn features-all" onClick={() => setAll((a) => !a)}>
              {all ? 'The five largest' : `See all ${features.length} →`}
            </button>
          )}
        </>
      )}
    </div>
  )
}
