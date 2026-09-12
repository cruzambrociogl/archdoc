import { useState } from 'react'
import { marked } from 'marked'
import type { DocsResponse } from './api'
import { editorLink, useApi } from './api'
import { Failure, Loading } from './Prov'

const escape = (s: string) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')

// Generated documents quote names from other people's configuration. Raw HTML in them is shown
// as text, never interpreted.
marked.use({ renderer: { html: (token) => escape(token.text) } })

const generated = (f: string) => /\.(generated\.md|svg|mmd)$/.test(f)

/** The generated documents, previewed; the human sections, linked and never opened. */
export function Docs() {
  const list = useApi<DocsResponse>('/api/docs')
  const [open, setOpen] = useState<string | null>(null)
  if (list.error) return <Failure error={list.error} />
  if (!list.data) return <Loading />
  const { dir } = list.data
  const files = list.data.generated ?? []
  const current = open ?? files[0] ?? null

  return (
    <div className="split">
      <section>
        <h2>Generated</h2>
        <p className="muted small">archdoc writes these and overwrites them on every run.</p>
        <ul className="plain">
          {files.map((f) => (
            <li key={f}>
              <button className={`link ${f === current ? 'strong' : ''}`} onClick={() => setOpen(f)}>
                {f}
              </button>
            </li>
          ))}
        </ul>
        <h2>Yours</h2>
        <p className="muted small">archdoc created these once and never reads them. They open in your editor.</p>
        <ul className="plain">
          {(list.data.human ?? []).map((h) => (
            <li key={h.file}>
              <a href={editorLink(dir, h.file)}>
                {h.number}. {h.title}
              </a>
            </li>
          ))}
        </ul>
      </section>
      <section>{current && <Preview name={current} dir={dir} onOpen={setOpen} />}</section>
    </div>
  )
}

function Preview({ name, dir, onOpen }: { name: string; dir: string; onOpen: (f: string) => void }) {
  const isSvg = name.endsWith('.svg')
  const doc = useApi<string>(isSvg ? null : `/api/docs/${encodeURIComponent(name)}`, 'text')

  if (isSvg) return <img className="doc-svg" src={`/api/docs/${encodeURIComponent(name)}`} alt={name} />
  if (doc.error) return <Failure error={doc.error} />
  if (doc.data === undefined) return <Loading />
  if (name.endsWith('.mmd')) return <pre>{doc.data}</pre>

  // Relative references point beside the document: generated files through the API, human
  // sections to the editor.
  const html = (marked.parse(doc.data, { async: false }) as string).replace(
    /(src|href)="(?![a-z]+:|#|\/)([^"]+)"/g,
    (_, attr: string, ref: string) =>
      `${attr}="${generated(ref) ? `/api/docs/${encodeURIComponent(ref)}` : editorLink(dir, ref)}"`,
  )

  return (
    <article
      className="markdown"
      onClick={(e) => {
        const a = (e.target as Element).closest('a')
        const href = a?.getAttribute('href') ?? ''
        if (href.startsWith('/api/docs/') && href.endsWith('.md')) {
          e.preventDefault()
          onOpen(decodeURIComponent(href.slice('/api/docs/'.length)))
        }
      }}
      dangerouslySetInnerHTML={{ __html: html }}
    />
  )
}
