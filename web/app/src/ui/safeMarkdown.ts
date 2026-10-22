// Markdown from a repository, made safe to put in the page. The documents archdoc writes carry
// strings from someone else's code — names, route summaries, a manifest's description — and the
// app renders the same documents in every place, the hosted website among them, where a script in
// a repository must never run (decisions, 22 Oct: the app renders hosted repositories itself).
// Two layers: raw HTML in the Markdown is shown as text, never interpreted; and what Marked builds
// is then sanitised by DOMPurify — no scripts, no event handlers, no javascript: links, no forms or
// frames — which also covers what the first cannot, a link's target.

import DOMPurify from 'dompurify'
import { marked } from 'marked'

const escape = (s: string) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
marked.use({ renderer: { html: (token) => escape(token.text) } })

const forbid = ['style', 'form', 'input', 'button', 'textarea', 'select', 'iframe', 'object', 'embed', 'base', 'meta', 'link']

/** Sanitises HTML that was built from a repository's text. data-* attributes the app adds survive. */
export function safeHtml(html: string): string {
  return DOMPurify.sanitize(html, { FORBID_TAGS: forbid, FORBID_ATTR: ['style'], ALLOW_DATA_ATTR: true })
}

/** Markdown to HTML, then sanitised. `rewrite` adjusts links before sanitising, never after. */
export function safeMarkdown(markdown: string, rewrite?: (html: string) => string): string {
  const html = marked.parse(markdown, { async: false }) as string
  return safeHtml(rewrite ? rewrite(html) : html)
}
