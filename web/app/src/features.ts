// What it does, without AI (platform-surface §6.6): never a list of route counts. A template turns
// each area's routes into verbs — POST, GET, DELETE /assets → "Adds, lists and deletes assets" —
// and every sentence carries the entries it was written from, so each is cited. Pages, background
// jobs and commands get a sentence of their own. Deterministic: the same model, the same sentences.

import type { Entry } from './api'

export interface Feature {
  key: string
  sentence: string
  entries: Entry[]
}

const verbOrder = ['adds', 'lists', 'shows', 'changes', 'deletes'] as const
type Verb = (typeof verbOrder)[number]

const isParam = (s: string) => s.startsWith(':') || s.startsWith('{') || s.startsWith('[') || s.startsWith('<') || s.startsWith('*')
const isNoise = (s: string) => s === 'api' || /^v\d+$/.test(s) || s === ''
const words = (s: string) => s.replace(/[-_.]+/g, ' ').trim()
const capital = (s: string) => s.charAt(0).toUpperCase() + s.slice(1)

function join(xs: string[]): string {
  return xs.length <= 1 ? (xs[0] ?? '') : `${xs.slice(0, -1).join(', ')} and ${xs[xs.length - 1]}`
}

/** "a, b, c and 4 more". */
function some(xs: string[], n = 3): string {
  return xs.length <= n + 1 ? join(xs) : `${xs.slice(0, n).join(', ')} and ${xs.length - n} more`
}

/** The path's meaningful segments: no api prefix, no version, no parameters' names. */
function segments(path: string): { area?: string; rest: string[]; param: boolean } {
  const all = path.split('?')[0].split('/').filter((s) => !isNoise(s))
  const area = all.find((s) => !isParam(s))
  const after = area ? all.slice(all.indexOf(area) + 1) : []
  return { area, rest: after.filter((s) => !isParam(s)), param: after.some(isParam) }
}

/** What an HTTP route does to its area: a verb, or — for POST /auth/login — an action of its own. */
function verbOf(e: Entry): { verb?: Verb; action?: string } {
  const { rest, param } = segments(e.path)
  const m = e.method.toUpperCase()
  if (rest.length > 0 && (m === 'POST' || m === 'PUT' || m === 'PATCH')) return { action: words(rest[rest.length - 1]) }
  if (rest.length > 0 && m === 'GET') return { action: words(rest.join(' ')) }
  if (m === 'POST') return { verb: 'adds' }
  if (m === 'GET') return { verb: param ? 'shows' : 'lists' }
  if (m === 'PUT' || m === 'PATCH') return { verb: 'changes' }
  if (m === 'DELETE') return { verb: 'deletes' }
  return {}
}

/** The features of a model's entries, the largest first. */
export function featuresOf(entries: Entry[]): Feature[] {
  const out: Feature[] = []

  const byArea = new Map<string, Entry[]>()
  for (const e of entries.filter((x) => x.kind === 'http')) {
    const area = segments(e.path).area
    if (!area) continue
    byArea.set(area, [...(byArea.get(area) ?? []), e])
  }
  for (const [area, es] of byArea) {
    const verbs = new Set<Verb>()
    const actions: string[] = []
    for (const e of es) {
      const v = verbOf(e)
      if (v.verb) verbs.add(v.verb)
      if (v.action && !actions.includes(v.action)) actions.push(v.action)
    }
    const noun = words(area)
    const doing = verbOrder.filter((v) => verbs.has(v))
    const parts: string[] = []
    if (doing.length) parts.push(`${capital(join(doing))} ${noun}`)
    if (actions.length) parts.push(doing.length ? `also ${some(actions)}` : `${capital(noun)}: ${some(actions)}`)
    if (parts.length) out.push({ key: `area:${area}`, sentence: `${parts.join(' — ')}.`, entries: es })
  }

  const named = (kind: string) => entries.filter((e) => e.kind === kind)
  // A page by its last meaningful segment: not a framework's file (+page.svelte, page.tsx, index) or a
  // route group's folder ((user)); the root is the home page.
  const framework = (s: string) => s.startsWith('+') || /^\(.*\)$/.test(s) || /^(page|index|layout)\.\w+$/.test(s)
  const label = (e: Entry) => {
    const last = e.path.split('/').filter((s) => s && !isParam(s) && !framework(s)).pop()
    return last ? words(last) : 'the home page'
  }
  const pages = named('page')
  if (pages.length) {
    const names = uniq(pages.map(label))
    const home = names.includes('the home page')
    const rest = names.filter((n) => n !== 'the home page')
    const sentence = home ? (rest.length ? `Has a home page, and pages for ${some(rest)}.` : 'Has a home page.') : `Has pages for ${some(rest)}.`
    out.push({ key: 'pages', sentence, entries: pages })
  }
  const jobs = named('job')
  if (jobs.length) out.push({ key: 'jobs', sentence: `Runs in the background: ${some(uniq(jobs.map((j) => j.path || j.handler)))}.`, entries: jobs })
  const commands = named('command')
  if (commands.length) out.push({ key: 'commands', sentence: `Offers commands: ${some(uniq(commands.map((c) => c.path || c.handler)))}.`, entries: commands })

  return out.sort((a, b) => b.entries.length - a.entries.length || a.key.localeCompare(b.key))
}

function uniq(xs: string[]): string[] {
  return [...new Set(xs.filter(Boolean))]
}
