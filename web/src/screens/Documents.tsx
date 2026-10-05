import { useState } from 'react'
import { marked } from 'marked'
import type { CompletenessResponse, DocsResponse } from '../api'
import { editorLink, useApi, when } from '../api'
import type { Route } from '../route'
import { Eyebrow, Failure, Loading } from '../ui/marks'

const escape = (s: string) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')

// Generated documents quote names from other people's configuration: raw HTML in them is shown as
// text, never interpreted. Comments are archdoc's own stamps, not content.
marked.use({ renderer: { html: (token) => escape(token.text) } })

type State = 'generated' | 'not started' | 'written' | 'may be stale' | 'missing'

interface Entry {
  number?: number
  title: string
  file: string
  state: State
  modified?: string
}

const isGenerated = (f: string) => /\.generated\.md$/.test(f)

/** "05-building-block-view.generated.md" → [5, "Building block view"] */
function fromFile(f: string): [number | undefined, string] {
  const m = f.match(/^(\d+)-(.+?)\.generated\.md$/)
  if (!m) return [undefined, f]
  const words = m[2].replace(/-/g, ' ')
  return [Number(m[1]), words.charAt(0).toUpperCase() + words.slice(1)]
}

const stateNote: Record<State, string> = {
  generated: 'archdoc writes this and overwrites it on every run.',
  'not started': 'Still the stub archdoc created.',
  written: 'Someone has written in it since the stub.',
  'may be stale': 'Written before the architecture last changed — it may describe a system that no longer exists.',
  missing: 'The file is gone. The next generate recreates the stub.',
}

/**
 * The arc42 reader (surface-spec §5.10): the documentation as it is committed. Generated sections
 * render here; the sections a person owns are never opened (OUT-03) — the app shows what the file
 * system reports about them, and, for one nobody has started, the questions this system raises,
 * regenerated from the model.
 */
export function Documents(props: { version: number | null; route: Route; go: (r: Partial<Route>, o?: { keep?: boolean; replace?: boolean }) => void }) {
  const docs = useApi<DocsResponse>('/api/docs')
  const comp = useApi<CompletenessResponse>(`/api/completeness${props.version ? `?version=${props.version}` : ''}`)

  if (docs.error) return <Failure error={docs.error} />
  if (comp.error) return <Failure error={comp.error} />
  if (!docs.data || !comp.data) return <Loading />

  const gen = (docs.data.generated ?? []).filter((f) => f.endsWith('.md'))
  const sections: Entry[] = [
    ...gen
      .filter((f) => /^\d/.test(f))
      .map((f): Entry => {
        const [number, title] = fromFile(f)
        return { number, title, file: f, state: 'generated' }
      }),
    ...comp.data.sections.map((s): Entry => ({ number: s.number, title: s.title, file: s.file, state: s.state as State, modified: s.modified })),
  ].sort((a, b) => (a.number ?? 99) - (b.number ?? 99))
  const extras: Entry[] = gen
    .filter((f) => !/^\d/.test(f))
    .map((f) => ({ title: f.startsWith('index') ? 'Index' : f.startsWith('coverage') ? 'Coverage — what archdoc could not see' : f, file: f, state: 'generated' }))

  const open = props.route.doc ?? extras.find((e) => e.file.startsWith('index'))?.file ?? sections[0]?.file
  const entry = [...extras, ...sections].find((e) => e.file === open)
  const count = (s: State) => sections.filter((e) => e.state === s).length

  return (
    <div className="documents">
      <aside className="doc-rail">
        <div className="doc-rail-head">
          <Eyebrow>arc42 · docs/architecture/</Eyebrow>
          <div className="doc-meter" aria-hidden="true">
            {sections.map((e) => (
              <span key={e.file} className={`seg seg-${e.state.replace(/ /g, '-')}`} />
            ))}
          </div>
          <div className="doc-counts">
            {count('generated')} generated · {count('written')} written · {count('may be stale')} may be stale · {count('not started')} not started
          </div>
        </div>
        {extras.map((e) => (
          <DocItem key={e.file} e={e} on={e.file === open} onClick={() => props.go({ screen: 'docs', doc: e.file })} />
        ))}
        <div className="doc-rail-rule" />
        {sections.map((e) => (
          <DocItem key={e.file} e={e} on={e.file === open} onClick={() => props.go({ screen: 'docs', doc: e.file })} />
        ))}
      </aside>

      <div className="doc-main">
        {entry ? (
          entry.state === 'generated' ? (
            <Generated entry={entry} onOpen={(f) => props.go({ screen: 'docs', doc: f })} dir={docs.data.dir} />
          ) : (
            <Yours entry={entry} dir={docs.data.dir} changed={comp.data.architecture_changed} version={props.version} />
          )
        ) : (
          <p className="muted">No documents yet — run archdoc generate.</p>
        )}
      </div>
    </div>
  )
}

function DocItem({ e, on, onClick }: { e: Entry; on: boolean; onClick: () => void }) {
  return (
    <button className={`doc-item ${on ? 'on' : ''}`} onClick={onClick}>
      <span className="doc-num">{e.number ?? ''}</span>
      <span className="doc-text">
        <span className="doc-title">{e.title}</span>
        <span className="doc-state">
          <span className={`state-shape shape-${e.state.replace(/ /g, '-')}`} />
          {e.state === 'generated' ? 'generated' : `yours · ${e.state}`}
        </span>
      </span>
    </button>
  )
}

function Head({ entry, owner }: { entry: Entry; owner: string }) {
  return (
    <>
      <div className="doc-head">
        <span className="mono muted small">docs/architecture/{entry.file}</span>
        <span className="owner-chip">{owner}</span>
      </div>
      <h1 className="page-title small">
        {entry.number ? `${entry.number}. ` : ''}
        {entry.title}
      </h1>
    </>
  )
}

function Generated({ entry, dir, onOpen }: { entry: Entry; dir: string; onOpen: (f: string) => void }) {
  const doc = useApi<string>(`/api/docs/${encodeURIComponent(entry.file)}`, 'text')
  if (doc.error) return <Failure error={doc.error} />
  if (doc.data === undefined) return <Loading />

  const source = doc.data.replace(/<!--[\s\S]*?-->/g, '')
  // Relative references point beside the document: generated files through the API (and
  // generated pages within this screen), the sections a person owns to their editor.
  const html = (marked.parse(source, { async: false }) as string).replace(
    /(src|href)="(?![a-z]+:|#|\/)([^"]+)"/g,
    (_, attr: string, ref: string) => {
      if (attr === 'href' && isGenerated(ref)) return `href="#screen=docs&doc=${encodeURIComponent(ref)}" data-doc="${ref}"`
      if (/\.generated\.md$|\.svg$|\.mmd$/.test(ref)) return `${attr}="/api/docs/${encodeURIComponent(ref)}"`
      return `${attr}="${editorLink(dir, ref)}"`
    },
  )

  return (
    <article className="doc-reading">
      <Head entry={entry} owner="Generated" />
      <div
        className="markdown-body"
        onClick={(e) => {
          const a = (e.target as Element).closest('a[data-doc]')
          if (a) {
            e.preventDefault()
            onOpen(a.getAttribute('data-doc')!)
          }
        }}
        dangerouslySetInnerHTML={{ __html: html }}
      />
      <p className="doc-foot">{stateNote.generated} Diagrams here are the committed SVGs; the explorer draws the same scene interactively.</p>
    </article>
  )
}

function Yours({ entry, dir, changed, version }: { entry: Entry; dir: string; changed: string; version: number | null }) {
  const q = useApi<{ markdown: string }>(
    entry.state === 'not started' && entry.number ? `/api/questions?section=${entry.number}${version ? `&version=${version}` : ''}` : null,
  )
  const [copied, setCopied] = useState(false)
  const link = editorLink(dir, entry.file)

  return (
    <article className="doc-reading">
      <Head entry={entry} owner={`Yours · ${entry.state}`} />
      <div className="yours-card">
        {entry.state === 'not started' ? (
          <>
            <p className="yours-lede">
              This section is yours. archdoc created the file once and never reads or writes it again. It has not been started,
              so here are the questions this system raises — regenerated from the model, not read from the file.
            </p>
            <Eyebrow>Questions</Eyebrow>
            {q.data ? (
              <div className="markdown-body questions" dangerouslySetInnerHTML={{ __html: marked.parse(q.data.markdown, { async: false }) as string }} />
            ) : q.error ? (
              <Failure error={q.error} />
            ) : (
              <Loading />
            )}
          </>
        ) : (
          <>
            <div className="yours-facts">
              <span className="fact-key">state</span>
              <span>{entry.state}</span>
              {entry.modified && (
                <>
                  <span className="fact-key">last edited</span>
                  <span>{when(entry.modified)}</span>
                </>
              )}
              {entry.state === 'may be stale' && (
                <>
                  <span className="fact-key">why stale</span>
                  <span>The architecture changed on {when(changed)}, after this file was last edited.</span>
                </>
              )}
            </div>
            <p className="yours-note">
              No preview. This section is yours, and archdoc never reads it (hard rule 2). The app shows only what the file
              system reports — when it last changed — and compares that with the architecture's history.
            </p>
          </>
        )}
        <div className="yours-actions">
          {entry.state !== 'missing' && (
            <a className="btn btn-ink" href={link}>
              Open in editor
            </a>
          )}
          {entry.state === 'not started' && q.data && (
            <button
              className="btn btn-outline"
              onClick={async () => {
                try {
                  await navigator.clipboard.writeText(q.data!.markdown)
                  setCopied(true)
                  setTimeout(() => setCopied(false), 1500)
                } catch {
                  /* clipboard refused */
                }
              }}
            >
              {copied ? 'Copied' : 'Copy questions'}
            </button>
          )}
          <span className="mono muted small">{stateNote[entry.state]}</span>
        </div>
      </div>
    </article>
  )
}
