// What archdoc knows: the sentences, the highlight and the lists, on Immich's coverage as the engine
// reports it (its numbers, a few of its rows), and on the cases Immich does not have — an application
// archdoc has no reader for, words the AI wrote.

import { test } from 'node:test'
import assert from 'node:assert/strict'
import { aiHighlight, aiSentence, aiWritten, readHighlight, readOf, readSentence, unreadRows, unresolvedRows } from '../src/screens/trust/facts.ts'

const at = (file: string, line: number) => ({ file, line })
const code = (app: string, container: string, files: number, lines: number, partial: string[] = [], extra: object = {}) => ({
  app, container, language: 'TypeScript', read: true, root: app, files, lines, components: 1, imports: {}, unresolved: [], partial, ...extra,
})
const call = (container: string, what: string, file: string, line: number) => ({
  container, what, note: 'an HTTP call whose address is computed at run time; nothing in the code names its target', provenance: at(file, line),
})

const immich = {
  source: 'docker/docker-compose.yml',
  read: [
    { file: 'docker/docker-compose.yml', used: true, why: 'selected' },
    { file: 'docker/docker-compose.dev.yml', used: false, why: 'not selected — deployable — 6 services with an image or build' },
  ],
  known: [], complete: 8, items: 32, traced: 2345, of: 2345, gaps: [], unconnected: [], limits: [],
  code: [
    code('machine-learning', 'svc:immich-machine-learning', 25, 2475),
    code('mobile', 'app:mobile', 649, 68844, [], { language: 'Dart', unresolved: [{ spec: 'package:immich_mobile/x.dart', provenance: at('mobile/lib/a.dart', 3) }] }),
    code('packages/cli', 'app:packages/cli', 6, 1175),
    code('server', 'svc:immich-server', 449, 58473, ['server/src/main.ts', 'server/src/b.ts', 'server/src/c.ts', 'server/src/d.ts', 'server/src/e.ts', 'server/src/f.ts', 'server/src/g.ts']),
    code('web', 'app:web', 632, 59013, Array.from({ length: 26 }, (_, i) => `web/src/f${i}.svelte`)),
  ],
  unresolved: [
    call('app:packages/cli', 'fetch(`${baseUrl}/assets`)', 'packages/cli/src/commands/asset.ts', 432),
    call('app:packages/cli', 'fetch(wellKnownUrl)', 'packages/cli/src/utils.ts', 72),
    ...['a', 'b', 'c', 'd', 'e'].map((x, i) => call('svc:immich-server', `fetch(${x})`, 'server/src/repositories/x.ts', 100 + i)),
    ...['a', 'b', 'c', 'd'].map((x, i) => call('app:web', `fetch(${x})`, 'web/src/lib/x.ts', 10 + i)),
  ],
} as never

const names: Record<string, string> = {
  'app:mobile': 'immich_mobile', 'app:packages/cli': '@immich/cli', 'app:web': 'immich-web',
  'svc:immich-server': 'immich-server', 'svc:immich-machine-learning': 'immich-machine-learning',
}
const name = (id: string) => names[id] ?? id

test('how much of each application was read', () => {
  assert.deepEqual(Object.fromEntries(readOf((immich as { code: never[] }).code)), {
    'svc:immich-machine-learning': 'full', 'app:mobile': 'full', 'app:packages/cli': 'full', 'svc:immich-server': 'partial', 'app:web': 'partial',
  })
})

test('the sentence for Immich: all read, all traced, eleven calls and two parts read in part', () => {
  assert.equal(
    readSentence(immich, name),
    'archdoc read the code of all 5 applications: 1,761 files, 189,980 lines. ' +
      'Every box and arrow here is traced to the file that proves it. ' +
      '11 calls go to an address set at run time, which archdoc couldn’t follow: 5 in immich-server, 4 in immich-web and 2 in @immich/cli. ' +
      'Some files were parsed only in part: 7 in immich-server and 26 in immich-web.',
  )
})

test('an application with no reader is said, and not counted as read', () => {
  const c = { ...(immich as object), code: [code('server', 'svc:api', 10, 100), { ...code('android', 'app:android', 0, 0), read: false, language: 'Kotlin' }], unresolved: [], traced: 10, of: 12 } as never
  assert.equal(
    readSentence(c, name),
    'archdoc read the code of 1 of its 2 applications: 10 files, 100 lines. It has no reader for Kotlin yet. 2 of the 12 boxes and arrows are not traced to a file.',
  )
  assert.equal(readOf((c as { code: never[] }).code).get('app:android'), 'unread')
  const h = readHighlight(c, ['svc:api', 'app:android'])
  assert.equal(h.tags?.get('app:android'), 'Not read')
  assert.ok(h.hatched?.has('app:android') && !h.hatched?.has('svc:api'))
  const row = unreadRows(c)[0]
  assert.equal(row.kind, 'not read')
  assert.equal(row.what, 'android/ · not read')
})

test('a report made before the traced count was kept says nothing about it, rather than guess', () => {
  const old = { ...(immich as object), traced: undefined, of: undefined, unresolved: [], code: [] } as never
  assert.equal(readSentence(old, name), 'archdoc read how this repository is configured; it has no application code archdoc could read.')
})

test('a repository with only its configuration', () => {
  const c = { ...(immich as object), code: [], unresolved: [] } as never
  assert.equal(readSentence(c, name), 'archdoc read how this repository is configured; it has no application code archdoc could read. Every box and arrow here is traced to the file that proves it.')
})

test('the highlight of what was read: nothing dimmed, what was read in part hatched and tagged', () => {
  const ids = ['actor:user', 'app:web', 'svc:immich-server', 'svc:database']
  const h = readHighlight(immich, ids)
  assert.deepEqual([...h.lit], ids)
  assert.equal(h.tone, 'muted')
  assert.deepEqual(Object.fromEntries(h.tags!), { 'svc:immich-server': 'Partly read', 'app:web': 'Partly read' })
  assert.deepEqual([...h.hatched!].sort(), ['app:web', 'svc:immich-server'])
})

test('what couldn’t be resolved: every call and import, in file order, each with its line', () => {
  const rows = unresolvedRows(immich)
  assert.equal(rows.length, 12)
  assert.deepEqual(rows.map((r) => `${r.cite.file}:${r.cite.line}`).slice(0, 4), ['mobile/lib/a.dart:3', 'packages/cli/src/commands/asset.ts:432', 'packages/cli/src/utils.ts:72', 'server/src/repositories/x.ts:100'])
  assert.equal(rows[0].what, 'import package:immich_mobile/x.dart')
  assert.equal(rows[0].box, 'app:mobile')
  // Line 9 before line 10, not as text.
  const order = unresolvedRows({ ...(immich as object), code: [], unresolved: [call('a', 'x', 'f.ts', 10), call('a', 'y', 'f.ts', 9)] } as never)
  assert.deepEqual(order.map((r) => r.cite.line), [9, 10])
})

test('what wasn’t read: files parsed in part, by application, then files passed over', () => {
  const rows = unreadRows(immich)
  assert.deepEqual(rows.map((r) => [r.kind, r.box ?? '', r.what]), [
    ['partly read', 'svc:immich-server', '7 files of 449 parsed only in part'],
    ['partly read', 'app:web', '26 files of 632 parsed only in part'],
    ['passed over', '', 'docker/docker-compose.dev.yml'],
  ])
  assert.equal(rows[1].files?.length, 26)
})

const model = (over: object = {}) =>
  ({
    name: 'x', source: '', edges: [{ from: 'a', to: 'b', provenance: null }, { from: 'b', to: 'c', label: 'calls', label_provenance: { origin: 'model', file: '', line: 0 }, provenance: null }],
    nodes: [
      { id: 'a', name: 'a', kind: 'application', evidence: 'declared', provenance: at('c.yml', 1) },
      { id: 'b', name: 'b', kind: 'application', evidence: 'declared', description: 'Serves photos', description_provenance: { origin: 'model', file: '', line: 0 }, provenance: at('c.yml', 2) },
      { id: 'c', name: 'c', kind: 'datastore', evidence: 'declared', provenance: at('c.yml', 3) },
    ],
    ...over,
  }) as never

test('which words the AI wrote: a description, an explanation of a box drawn here, an arrow’s words', () => {
  const m = model({ explanations: [{ element: 'a', claims: [], fingerprint: '', provenance: at('x', 1) }, { element: 'cmp:not-drawn', claims: [], fingerprint: '', provenance: at('x', 1) }] })
  const w = aiWritten(m)
  assert.deepEqual([...w.boxes].sort(), ['a', 'b'])
  assert.equal(w.arrows, 1)
  assert.equal(
    aiSentence(m),
    'The boxes and the arrows are proven. What 2 of the 3 boxes say about themselves was written by AI from what archdoc read, and is marked. The words on 1 arrow were written by AI too, and are marked.',
  )
  const h = aiHighlight(m)
  assert.equal(h.tone, 'interp')
  assert.deepEqual(Object.fromEntries(h.tags!), { a: 'Written by AI', b: 'Written by AI' })
  assert.equal(h.lit.size, 3)
})

test('when the AI wrote nothing, it says so', () => {
  const m = model({ nodes: [{ id: 'a', name: 'a', kind: 'application', evidence: 'declared', provenance: at('c.yml', 1) }], edges: [] })
  assert.equal(aiSentence(m), 'None of it. Every box, arrow and word on this diagram comes from the code, its configuration, or archdoc’s catalog of known technologies.')
  assert.equal(aiHighlight(m).tags?.size, 0)
})

test('one box written, the arrows from a rule', () => {
  const m = model({ edges: [] })
  assert.equal(aiSentence(m), 'The boxes and the arrows are proven. What 1 of the 3 boxes says about itself was written by AI from what archdoc read, and is marked. The words on the arrows come from a rule, not from AI.')
})
