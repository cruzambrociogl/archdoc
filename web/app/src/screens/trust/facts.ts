// What archdoc knows (platform-surface §17.3; app-design-screens §14): the system drawn by how much
// of it archdoc read, and by which words the AI wrote. Every value is the coverage report's or the
// model's own; this only counts and says them. Pure: the screen draws what this returns, and the
// tests read it.

import type { CodeRead, CoverageResponse, Model, Provenance } from '../../api'
import type { Highlight } from '../explorer/scene'

/** How much of one application archdoc read. */
export type Read = 'full' | 'partial' | 'unread'

/** Each application's code, by the box it is: read in full, read with files parsed only in part, or not read. */
export function readOf(code: CodeRead[]): Map<string, Read> {
  const out = new Map<string, Read>()
  for (const c of code) if (c.container) out.set(c.container, !c.read ? 'unread' : c.partial.length ? 'partial' : 'full')
  return out
}

/** "A", "A and B", "A, B and C". */
export function and(xs: string[]): string {
  if (xs.length <= 1) return xs[0] ?? ''
  return `${xs.slice(0, -1).join(', ')} and ${xs[xs.length - 1]}`
}

const n = (x: number) => x.toLocaleString('en-US')
const plural = (k: number, one: string, many: string) => `${n(k)} ${k === 1 ? one : many}`

/** Counts by box, largest first, then by name: "5 in immich-server, 4 in immich-web". */
function byBox(ids: string[], name: (id: string) => string): string {
  const count = new Map<string, number>()
  for (const id of ids) count.set(id, (count.get(id) ?? 0) + 1)
  return and(
    [...count]
      .sort((a, b) => b[1] - a[1] || name(a[0]).localeCompare(name(b[0])))
      .map(([id, k]) => `${n(k)} in ${name(id)}`),
  )
}

/**
 * The answer, as the band says it: what was read, that every box and arrow is traced to a file,
 * what could not be followed and where, and what was read only in part.
 */
export function readSentence(c: CoverageResponse, name: (id: string) => string): string {
  const code = c.code ?? []
  const read = code.filter((x) => x.read)
  const files = read.reduce((t, x) => t + x.files, 0)
  const lines = read.reduce((t, x) => t + x.lines, 0)
  const out: string[] = []

  if (!code.length) out.push('archdoc read how this repository is configured; it has no application code archdoc could read.')
  else if (read.length === code.length)
    out.push(`archdoc read the code of ${code.length === 1 ? 'its one application' : `all ${code.length} applications`}: ${plural(files, 'file', 'files')}, ${plural(lines, 'line', 'lines')}.`)
  else {
    const missing = [...new Set(code.filter((x) => !x.read).map((x) => x.language))].sort()
    out.push(
      `archdoc read the code of ${n(read.length)} of its ${n(code.length)} applications: ${plural(files, 'file', 'files')}, ${plural(lines, 'line', 'lines')}. It has no reader for ${and(missing)} yet.`,
    )
  }

  // A report from before it counted them as numbers says nothing here rather than a guess.
  if (c.traced !== undefined && c.of !== undefined && c.of > 0)
    out.push(c.traced === c.of ? 'Every box and arrow here is traced to the file that proves it.' : `${n(c.of - c.traced)} of the ${n(c.of)} boxes and arrows are not traced to a file.`)

  const calls = c.unresolved ?? []
  if (calls.length) out.push(`${plural(calls.length, 'call goes', 'calls go')} to an address set at run time, which archdoc couldn’t follow: ${byBox(calls.map((u) => u.container), name)}.`)

  const partial = code.filter((x) => x.read && x.partial.length && x.container)
  if (partial.length) out.push(`Some files were parsed only in part: ${and(partial.map((x) => `${n(x.partial.length)} in ${name(x.container)}`))}.`)

  return out.join(' ')
}

/** What was read, on the diagram: nothing dimmed; what was not read in full hatched and tagged. */
export function readHighlight(c: CoverageResponse, ids: string[]): Highlight {
  const reads = readOf(c.code ?? [])
  const tags = new Map<string, string>()
  const hatched = new Set<string>()
  for (const [id, r] of reads) {
    if (r === 'full') continue
    tags.set(id, r === 'partial' ? 'Partly read' : 'Not read')
    hatched.add(id)
  }
  return { lit: new Set(ids), tags, tone: 'muted', hatched }
}

/** One thing archdoc could not settle: where, what, why, and the line. */
export interface Loose {
  box: string
  what: string
  why: string
  cite: Provenance
}

/** What couldn't be resolved: calls to an address set at run time, then imports archdoc couldn't follow, in file order. */
export function unresolvedRows(c: CoverageResponse): Loose[] {
  const calls = (c.unresolved ?? []).map((u) => ({ box: u.container, what: u.what, why: u.note, cite: u.provenance }))
  const imports = (c.code ?? []).flatMap((x) =>
    x.unresolved.map((u) => ({ box: x.container, what: `import ${u.spec}`, why: 'an import archdoc could not follow to a file', cite: u.provenance })),
  )
  const at = (p: Provenance) => `${p.file}\u0000${String(p.line).padStart(8, '0')}`
  return [...calls, ...imports].sort((a, b) => at(a.cite).localeCompare(at(b.cite)))
}

/** One thing archdoc did not read, or read only in part. */
export interface Unread {
  kind: 'not read' | 'partly read' | 'passed over'
  /** The box it belongs to, where it belongs to one. */
  box?: string
  what: string
  why: string
  files?: string[]
}

/** What wasn't read: applications in a language with no reader, files parsed only in part, files read and passed over. */
export function unreadRows(c: CoverageResponse): Unread[] {
  const out: Unread[] = []
  for (const x of c.code ?? []) {
    if (!x.read)
      out.push({
        kind: 'not read',
        box: x.container,
        what: `${x.app}/ · not read`,
        why: `${x.language}: archdoc has no reader for it yet, so what is inside is not drawn.${x.schemas?.length ? ` Its tables are read from ${plural(x.schemas.length, 'schema file', 'schema files')}.` : ''}`,
      })
    else if (x.partial.length)
      out.push({
        kind: 'partly read',
        box: x.container,
        what: `${plural(x.partial.length, 'file', 'files')} of ${n(x.files)} parsed only in part`,
        why: `The parser stopped at something it could not read in ${x.partial.length === 1 ? 'this file' : 'these files'}; what came before it was read.`,
        files: x.partial,
      })
  }
  for (const f of c.read) if (!f.used) out.push({ kind: 'passed over', what: f.file, why: `Read and passed over: ${f.why}.` })
  return out
}

/** Which words the AI wrote: the boxes with a value the model wrote, or an explanation of their own; and the arrows. */
export function aiWritten(m: Model): { boxes: Set<string>; arrows: number } {
  const boxes = new Set<string>()
  const byModel = (p?: Provenance) => p?.origin === 'model'
  for (const node of m.nodes ?? []) if (byModel(node.name_provenance) || byModel(node.description_provenance) || byModel(node.technology_provenance)) boxes.add(node.id)
  const drawn = new Set((m.nodes ?? []).map((x) => x.id))
  for (const x of m.explanations ?? []) if (drawn.has(x.element)) boxes.add(x.element)
  const arrows = (m.edges ?? []).filter((e) => byModel(e.label_provenance)).length
  return { boxes, arrows }
}

/** The answer for the AI's words, as the band says it. */
export function aiSentence(m: Model): string {
  const { boxes, arrows } = aiWritten(m)
  const all = (m.nodes ?? []).length
  if (!boxes.size && !arrows) return 'None of it. Every box, arrow and word on this diagram comes from the code, its configuration, or archdoc’s catalog of known technologies.'
  const out = ['The boxes and the arrows are proven.']
  if (boxes.size) {
    const who = boxes.size === all ? 'every box says about itself' : `${n(boxes.size)} of the ${n(all)} boxes ${boxes.size === 1 ? 'says about itself' : 'say about themselves'}`
    out.push(`What ${who} was written by AI from what archdoc read, and is marked.`)
  }
  out.push(arrows ? `The words on ${plural(arrows, 'arrow were', 'arrows were')} written by AI too, and are marked.` : 'The words on the arrows come from a rule, not from AI.')
  return out.join(' ')
}

/** The AI's words, on the diagram: nothing dimmed; every box it wrote about tagged in its blue. */
export function aiHighlight(m: Model): Highlight {
  const { boxes } = aiWritten(m)
  return { lit: new Set((m.nodes ?? []).map((x) => x.id)), tags: new Map([...boxes].map((id) => [id, 'Written by AI'])), tone: 'interp' }
}
