import type { Origin, Provenance } from '../api'
import { useEditorLink } from '../api'

const origins: Record<Origin, { title: string }> = {
  extraction: { title: 'Read from a file in the repository' },
  catalog: { title: "From archdoc's bundled catalog — deterministic, but not stated by the repository, so it records no line" },
  rules: { title: 'Asserted by a person in rules.yaml' },
  model: { title: 'Written by the language model — an interpretation on top of proven facts, never a fact' },
}

/** The origin glyph: filled = read by the machine, half = catalog, ring = a rule, diamond = the model. */
export function OriginGlyph({ origin }: { origin: Origin }) {
  return <span className={`glyph glyph-${origin}`} aria-hidden="true" />
}

/**
 * A citation (surface-spec §6.2): the origin glyph, then the file and line in the link colour.
 * The directory is muted and long paths truncate from the left, so the file and line always show.
 * Live, a click opens the editor at the line.
 */
export function Cite({ p, compact }: { p?: Provenance; compact?: boolean }) {
  const link = useEditorLink()
  if (!p || (!p.file && !p.note)) {
    return <span className="cite cite-missing">no source recorded</span>
  }
  const origin: Origin = p.origin || 'extraction'
  const title = `${origins[origin]?.title ?? origin}${p.note ? ` — ${p.note}` : ''}`
  const slash = p.file ? p.file.lastIndexOf('/') : -1
  const dir = slash >= 0 ? p.file.slice(0, slash + 1) : ''
  const base = slash >= 0 ? p.file.slice(slash + 1) : p.file
  const body = p.file ? (
    <span className="cite-ref">
      <bdi>
        {!compact && <span className="cite-dir">{dir}</span>}
        <span className="cite-file">{base}</span>
        {p.line ? <span className="cite-line">:{p.line}</span> : null}
      </bdi>
    </span>
  ) : (
    <span className="cite-ref cite-note">{p.note}</span>
  )

  const href = p.file ? link(p.file, p.line) : undefined
  return href ? (
    <a className="cite" href={href} title={`${title}\n${p.file}${p.line ? `:${p.line}` : ''} — open at the line`}>
      <OriginGlyph origin={origin} />
      {body}
    </a>
  ) : (
    <span className="cite" title={title}>
      <OriginGlyph origin={origin} />
      {body}
    </span>
  )
}
