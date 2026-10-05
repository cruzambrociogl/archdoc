import { useEffect, useMemo, useRef, useState } from 'react'
import type { DocsResponse, ModelResponse, Node, Provenance, SavedView } from '../api'
import { useApi } from '../api'
import type { Route, Screen } from '../route'
import { kindStyle } from '../ui/kinds'
import { TruthMark } from '../ui/marks'

type Go = (r: Partial<Route>) => void

interface Hit {
  group: 'Elements' | 'Relationships' | 'Files' | 'Documents' | 'Saved views' | 'Screens'
  key: string
  title: string
  meta: string
  cite?: string
  interpreted?: boolean
  go: () => void
  /** A file: what archdoc knows about it. */
  file?: { path: string; proves: string[] }
}

const screenNames: [Screen, string][] = [
  ['overview', 'Overview'],
  ['explorer', 'Explorer'],
  ['changes', 'Changes'],
  ['docs', 'Documents'],
  ['rules', 'Corrections'],
  ['runs', 'Network runs'],
]

const at = (p?: Provenance) => (p?.file ? `${p.file}${p.line ? `:${p.line}` : ''}` : undefined)

/**
 * Global search (surface-spec §5.17): one index over elements, relationships, the files that prove
 * them, documents, saved views and screens. Results are grouped by type and each carries its truth
 * state. Searching a file path answers "what does archdoc know about this file?" — the elements and
 * relationships it proves.
 */
export function Palette(props: { open: boolean; onClose: () => void; go: Go; views: SavedView[]; version: number | null }) {
  const [q, setQ] = useState('')
  const [cursor, setCursor] = useState(0)
  const input = useRef<HTMLInputElement>(null)
  const model = useApi<ModelResponse>(props.open ? `/api/model${props.version ? `?version=${props.version}` : ''}` : null)
  const docs = useApi<DocsResponse>(props.open ? '/api/docs' : null)

  useEffect(() => {
    if (props.open) {
      setQ('')
      setCursor(0)
      setTimeout(() => input.current?.focus(), 0)
    }
  }, [props.open])

  const index = useMemo(() => {
    if (!model.data) return [] as Hit[]
    const m = model.data
    const inContainer = new Set((m.container.nodes ?? []).map((n) => n.id))
    const nodes = m.model.nodes ?? []
    const name = (id: string) => nodes.find((n) => n.id === id)?.name ?? id
    const open = (n: Node) => props.go({ screen: 'explorer', level: inContainer.has(n.id) ? 'container' : 'context', focus: n.id })

    const hits: Hit[] = []
    for (const n of nodes) {
      hits.push({
        group: 'Elements',
        key: `n:${n.id}`,
        title: n.name,
        meta: [kindStyle(n.kind).label, n.technology, n.id].filter(Boolean).join(' · '),
        cite: at(n.provenance),
        go: () => open(n),
      })
    }
    for (const e of m.model.edges ?? []) {
      const level = inContainer.has(e.from) && inContainer.has(e.to) ? 'container' : 'context'
      hits.push({
        group: 'Relationships',
        key: `e:${e.from}>${e.to}`,
        title: `${name(e.from)} → ${name(e.to)}`,
        meta: [e.label, e.technology].filter(Boolean).join(' · ') || 'relationship',
        cite: at((e.provenance ?? [])[0]),
        interpreted: e.label_provenance?.origin === 'model',
        go: () => props.go({ screen: 'explorer', level, focus: `${e.from}>${e.to}` }),
      })
    }

    // Every file that proves something, and what it proves.
    const files = new Map<string, string[]>()
    const note = (p: Provenance | undefined, what: string) => {
      if (!p?.file) return
      files.set(p.file, [...(files.get(p.file) ?? []), what])
    }
    for (const n of nodes) note(n.provenance, n.name)
    for (const e of m.model.edges ?? []) for (const p of e.provenance ?? []) note(p, `${name(e.from)} → ${name(e.to)}`)
    for (const [path, proves] of [...files].sort(([a], [b]) => a.localeCompare(b))) {
      hits.push({
        group: 'Files',
        key: `f:${path}`,
        title: path,
        meta: `proves ${proves.length} ${proves.length === 1 ? 'thing' : 'things'}`,
        file: { path, proves: [...new Set(proves)].sort() },
        go: () => {},
      })
    }

    for (const f of (docs.data?.generated ?? []).filter((x) => x.endsWith('.md'))) {
      hits.push({ group: 'Documents', key: `d:${f}`, title: f, meta: 'generated', go: () => props.go({ screen: 'docs', doc: f }) })
    }
    for (const h of docs.data?.human ?? []) {
      hits.push({ group: 'Documents', key: `d:${h.file}`, title: `${h.number}. ${h.title}`, meta: `yours · ${h.file}`, go: () => props.go({ screen: 'docs', doc: h.file }) })
    }
    for (const v of props.views) {
      hits.push({
        group: 'Saved views',
        key: `v:${v.name}`,
        title: v.name,
        meta: [v.level, v.focus, v.find && `“${v.find}”`].filter(Boolean).join(' · '),
        go: () => props.go({ screen: 'explorer', level: v.level, focus: v.focus, q: v.find, dim: v.dim ? '1' : undefined }),
      })
    }
    for (const [id, label] of screenNames) hits.push({ group: 'Screens', key: `s:${id}`, title: label, meta: 'screen', go: () => props.go({ screen: id }) })
    return hits
  }, [model.data, docs.data, props.views, props.go])

  const query = q.trim().toLowerCase()
  const results = useMemo(() => {
    if (!query) return [] as Hit[]
    const scored = index
      .map((h) => {
        const t = h.title.toLowerCase()
        // Screens match on their name only: "screen" is in every one of their labels.
        const all = h.group === 'Screens' ? t : `${t} ${h.meta.toLowerCase()} ${(h.cite ?? '').toLowerCase()}`
        const score = t === query ? 0 : t.startsWith(query) ? 1 : t.includes(query) ? 2 : all.includes(query) ? 3 : -1
        return { h, score }
      })
      .filter((x) => x.score >= 0)
    const order = ['Elements', 'Relationships', 'Files', 'Documents', 'Saved views', 'Screens']
    scored.sort((a, b) => order.indexOf(a.h.group) - order.indexOf(b.h.group) || a.score - b.score || a.h.title.localeCompare(b.h.title))
    // At most eight a group, so one large group cannot hide the others.
    const per = new Map<string, number>()
    return scored
      .filter(({ h }) => {
        const n = per.get(h.group) ?? 0
        per.set(h.group, n + 1)
        return n < 8
      })
      .map((x) => x.h)
  }, [index, query])

  // A query that names a file exactly answers what archdoc knows about it.
  const file = results.find((h) => h.file && (h.title.toLowerCase() === query || h.title.toLowerCase().endsWith('/' + query)))?.file

  useEffect(() => setCursor(0), [query])

  if (!props.open) return null

  const pick = (h: Hit) => {
    // A file answers in place: what archdoc knows about it.
    if (h.file) {
      setQ(h.title)
      input.current?.focus()
      return
    }
    props.onClose()
    h.go()
  }
  const groups = [...new Set(results.map((h) => h.group))]
  const tries = index.filter((h) => h.group === 'Elements').slice(0, 2).map((h) => h.title)
  const firstFile = index.find((h) => h.group === 'Files')?.title

  return (
    <div className="dialog-scrim palette-scrim" onClick={props.onClose}>
      <div className="palette" role="dialog" aria-label="Search" onClick={(e) => e.stopPropagation()}>
        <div className="palette-input">
          <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6">
            <path d="M7 12A5 5 0 1 0 7 2a5 5 0 0 0 0 10z M10.5 10.5L14 14" />
          </svg>
          <input
            ref={input}
            value={q}
            placeholder="Search elements, relationships, files, documents…"
            onChange={(e) => setQ(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Escape') props.onClose()
              else if (e.key === 'ArrowDown') {
                e.preventDefault()
                setCursor((c) => Math.min(c + 1, results.length - 1))
              } else if (e.key === 'ArrowUp') {
                e.preventDefault()
                setCursor((c) => Math.max(c - 1, 0))
              } else if (e.key === 'Enter' && results[cursor]) pick(results[cursor])
            }}
          />
          <kbd>Esc</kbd>
        </div>

        {!query && (
          <div className="palette-try">
            <span>Try</span>
            {[...tries, firstFile].filter(Boolean).map((t) => (
              <button key={t} onClick={() => setQ(t!)}>
                {t}
              </button>
            ))}
          </div>
        )}

        {file && (
          <div className="palette-file">
            <div className="eyebrow">What archdoc knows about this file</div>
            <div className="mono">{file.path}</div>
            <div className="palette-file-proves">
              Proves {file.proves.length}: {file.proves.join(', ')}
            </div>
          </div>
        )}

        <div className="palette-results">
          {groups.map((g) => (
            <div key={g}>
              <div className="palette-group eyebrow">
                {g} · {results.filter((h) => h.group === g).length}
              </div>
              {results
                .filter((h) => h.group === g)
                .map((h) => {
                  const i = results.indexOf(h)
                  return (
                    <button key={h.key} className={`palette-row ${i === cursor ? 'on' : ''}`} onMouseEnter={() => setCursor(i)} onClick={() => pick(h)}>
                      <span className="palette-mark">
                        {(h.group === 'Elements' || h.group === 'Relationships') && <TruthMark state={h.interpreted ? 'interpreted' : 'proven'} />}
                      </span>
                      <span className="palette-text">
                        <span className={`palette-title ${h.interpreted ? 'interpreted' : ''}`}>{h.title}</span>
                        <span className="palette-meta">{h.meta}</span>
                      </span>
                      {h.cite && <span className="palette-cite">{h.cite}</span>}
                    </button>
                  )
                })}
            </div>
          ))}
          {query && !results.length && (
            <p className="palette-none">
              Nothing in the model matches “{q}”. archdoc only knows what it extracted; the coverage report lists what it could not
              see.
            </p>
          )}
        </div>
        <div className="palette-foot">
          <span>↑↓ move</span>
          <span>↵ open</span>
          <span>■ proven · ◇ interpreted</span>
        </div>
      </div>
    </div>
  )
}
