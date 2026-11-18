// One part: reach, its sentence, its list and its highlight, on Immich's Apps and services as the
// engine draws them (the arrows below are the model's own, provenance aside).

import { test } from 'node:test'
import assert from 'node:assert/strict'
import { and, highlight, reach, rows, sentence } from '../src/screens/part/reach.ts'

const p = (file: string, line: number) => [{ file, line }]
const edges = [
  { from: 'actor:user', to: 'app:mobile', label: 'uses', provenance: p('docker/docker-compose.yml', 1) },
  { from: 'actor:user', to: 'app:packages/cli', label: 'runs', provenance: p('docker/docker-compose.yml', 2) },
  { from: 'actor:user', to: 'app:web', label: 'uses', provenance: p('docker/docker-compose.yml', 3) },
  { from: 'actor:user', to: 'svc:immich-server', label: 'reaches', provenance: p('docker/docker-compose.yml', 4) },
  { from: 'app:mobile', to: 'svc:immich-server', label: 'calls the API of', technology: 'http', provenance: p('mobile/lib/api.dart', 10) },
  { from: 'app:packages/cli', to: 'svc:immich-server', label: 'calls the API of', technology: 'http', provenance: p('cli/src/api.ts', 11) },
  { from: 'app:web', to: 'svc:immich-server', label: 'calls the API of', technology: 'http', provenance: p('web/src/api.ts', 12) },
  { from: 'svc:immich-machine-learning', to: 'ext:hugging-face-hub', label: 'downloads models from', provenance: p('machine-learning/app.py', 13) },
  { from: 'svc:immich-server', to: 'ext:email-server-smtp', label: 'sends email through', provenance: p('server/src/email.ts', 14) },
  { from: 'svc:immich-server', to: 'svc:database', label: 'connects to', provenance: p('docker/docker-compose.yml', 15) },
  { from: 'svc:immich-server', to: 'svc:immich-machine-learning', label: 'calls', technology: 'http', provenance: p('docker/docker-compose.yml', 16) },
  { from: 'svc:immich-server', to: 'svc:redis', label: 'connects to', provenance: p('docker/docker-compose.yml', 17) },
] as never[]

const names: Record<string, string> = {
  'actor:user': 'User',
  'app:mobile': 'immich_mobile',
  'app:packages/cli': '@immich/cli',
  'app:web': 'immich-web',
  'svc:immich-server': 'immich-server',
  'svc:immich-machine-learning': 'immich-machine-learning',
  'svc:database': 'database',
  'svc:redis': 'redis',
  'ext:hugging-face-hub': 'Hugging Face Hub',
  'ext:email-server-smtp': 'Email server (SMTP)',
}
const name = (id: string) => names[id] ?? id

test('what a client touches: the server directly, everything after it through the server', () => {
  const r = reach(edges, 'app:mobile', 'touches')
  assert.deepEqual(
    r.map((x) => [x.id, x.via ?? '']),
    [
      ['svc:immich-server', ''],
      ['ext:email-server-smtp', 'svc:immich-server'],
      ['svc:database', 'svc:immich-server'],
      ['svc:immich-machine-learning', 'svc:immich-server'],
      ['svc:redis', 'svc:immich-server'],
      // Two steps away, still named by the first part on the way.
      ['ext:hugging-face-hub', 'svc:immich-server'],
    ],
  )
  assert.equal(r[0].edges.length, 1)
  assert.equal(r[0].edges[0].label, 'calls the API of')
  assert.deepEqual(r.slice(1).map((x) => x.edges.length), [0, 0, 0, 0, 0])
})

test('the sentence for what a client touches', () => {
  assert.equal(
    sentence('touches', 'app:mobile', reach(edges, 'app:mobile', 'touches'), name),
    'immich_mobile calls immich-server directly, and reaches database, Email server (SMTP), Hugging Face Hub, immich-machine-learning and redis through immich-server.',
  )
})

test('what depends on the database: the server directly, the clients and the user through it', () => {
  const r = reach(edges, 'svc:database', 'depends')
  assert.deepEqual(r.map((x) => [x.id, x.via ?? '']), [
    ['svc:immich-server', ''],
    ['actor:user', 'svc:immich-server'],
    ['app:mobile', 'svc:immich-server'],
    ['app:packages/cli', 'svc:immich-server'],
    ['app:web', 'svc:immich-server'],
  ])
  assert.equal(
    sentence('depends', 'svc:database', r, name),
    'immich-server uses database directly. @immich/cli, immich_mobile, immich-web and User reach it through immich-server.',
  )
})

test('what depends on the server: four directly, nothing further', () => {
  const r = reach(edges, 'svc:immich-server', 'depends')
  assert.deepEqual(r.map((x) => x.id).sort(), ['actor:user', 'app:mobile', 'app:packages/cli', 'app:web'])
  assert.ok(r.every((x) => !x.via))
  assert.equal(sentence('depends', 'svc:immich-server', r, name), '@immich/cli, immich_mobile, immich-web and User use immich-server directly.')
})

test('a part that calls nothing, and a part nothing uses', () => {
  assert.deepEqual(reach(edges, 'svc:database', 'touches'), [])
  assert.equal(sentence('touches', 'svc:database', [], name), 'database calls nothing on this diagram. It only answers.')
  assert.deepEqual(reach(edges, 'actor:user', 'depends'), [])
  assert.equal(sentence('depends', 'actor:user', [], name), 'Nothing on this diagram uses User.')
})

test('one arrow each way between two parts: the subject is never reached, and the walk ends', () => {
  const loop = [
    { from: 'a', to: 'b', label: 'calls', provenance: null },
    { from: 'b', to: 'a', label: 'calls back', provenance: null },
    { from: 'b', to: 'c', label: 'calls', provenance: null },
  ] as never[]
  assert.deepEqual(reach(loop, 'a', 'touches').map((x) => [x.id, x.via ?? '']), [['b', ''], ['c', 'b']])
  assert.deepEqual(reach(loop, 'a', 'depends').map((x) => [x.id, x.via ?? '']), [['b', '']])
})

test('every arrow between the subject and a part is kept, each with its words and proof', () => {
  const two = [
    { from: 'a', to: 'b', label: 'calls', technology: 'http', provenance: p('x.ts', 1) },
    { from: 'a', to: 'b', label: 'depends on', provenance: p('compose.yml', 2) },
  ] as never[]
  const r = reach(two, 'a', 'touches')
  assert.equal(r.length, 1)
  assert.deepEqual(r[0].edges.map((e: { label: string }) => e.label), ['calls', 'depends on'])
})

test('the list: direct first, then the rest, each by name', () => {
  const list = rows(reach(edges, 'svc:database', 'depends'), name).map((x) => name(x.id))
  assert.deepEqual(list, ['immich-server', '@immich/cli', 'immich_mobile', 'immich-web', 'User'])
})

test('the highlight: the subject and what it reaches lit, the indirect ones tagged', () => {
  const h = highlight('app:web', reach(edges, 'app:web', 'touches'), name)
  assert.ok(h.lit.has('app:web') && h.lit.has('svc:immich-server') && h.lit.has('svc:redis'))
  assert.ok(!h.lit.has('app:mobile') && !h.lit.has('actor:user'))
  assert.equal(h.tags?.get('svc:redis'), 'through immich-server')
  assert.equal(h.tags?.has('svc:immich-server'), false)
})

test('the same model gives the same answer', () => {
  const once = JSON.stringify(reach(edges, 'app:web', 'touches'))
  assert.equal(JSON.stringify(reach(edges, 'app:web', 'touches')), once)
})

test('and', () => {
  assert.equal(and([]), '')
  assert.equal(and(['a']), 'a')
  assert.equal(and(['a', 'b']), 'a and b')
  assert.equal(and(['a', 'b', 'c']), 'a, b and c')
})
