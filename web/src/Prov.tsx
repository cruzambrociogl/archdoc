import type { Provenance } from './api'
import { useEditorLink } from './api'

const origins: Record<string, { label: string; title: string }> = {
  extraction: { label: 'file', title: 'Read from a file in the repository' },
  catalog: { label: 'catalog', title: "Looked up from archdoc's image catalog — deterministic, but not stated by the repository" },
  rules: { label: 'your rule', title: 'Written by a person in rules.yaml' },
  model: { label: 'model', title: 'Suggested by the language model — an interpretation, never a fact' },
}

/** One citation: where a value came from, opening in the editor at the line. */
export function Prov({ p }: { p?: Provenance }) {
  const link = useEditorLink()
  if (!p || (!p.file && !p.note)) return <span className="prov unknown">no source recorded</span>
  const origin = p.origin || 'extraction'
  const o = origins[origin] ?? { label: origin, title: '' }
  return (
    <span className={`prov ${origin}`}>
      <span className="origin" title={o.title}>
        {o.label}
      </span>
      {p.file && (
        <a href={link(p.file, p.line)} title="Open in VS Code">
          {p.file}
          {p.line ? `:${p.line}` : ''}
        </a>
      )}
      {p.note && <span className="note">{p.note}</span>}
    </span>
  )
}

export function Loading() {
  return <p className="muted">Loading…</p>
}

export function Failure({ error }: { error: string }) {
  return <p className="error">{error}</p>
}
