// What breaks without one part: the band's sentences and the highlight, on the engine's answers for
// Immich (their shape and counts), and the cases Immich does not have.

import { test } from 'node:test'
import assert from 'node:assert/strict'
import { downHighlight, downSentence, ordered, stopsByKind } from '../src/screens/part/down.ts'

const at = { file: 'x.ts', line: 1 }
const stop = (entry: string, why = 'reads asset') => ({ entry, why, cite: at })
const many = (n: number, kind: string) => Array.from({ length: n }, (_, i) => stop(`${kind}:immich-server ${i}`))
const names: Record<string, string> = { 'svc:immich-server': 'immich-server', 'svc:database': 'database', 'svc:redis': 'redis', 'app:web': 'immich-web', 'app:mobile': 'immich_mobile', 'svc:ml': 'immich-machine-learning', 'actor:user': 'User' }
const name = (id: string) => names[id] ?? id
const dep = (o: object) => ({ id: 'svc:immich-server', how: 'reads and writes its tables', known: true, stops: [], unaffected: 0, unfollowed: 0, provenance: null, ...o })
const through = [{ id: 'app:web', via: 'svc:immich-server' }]

test('the database down: what stops, by kind, and what is not affected or not known', () => {
  const b = { part: 'svc:database', dependents: [dep({ stops: [...many(240, 'route'), ...many(62, 'job')], unaffected: 61, unfollowed: 17 })], through, basis: '' }
  assert.equal(
    downSentence(b, name),
    'immich-server stops 240 routes and 62 background jobs that read or write its tables. 61 of its ways in reach nothing of it; 17 weren’t followed, so can’t be told.',
  )
})

test('the queue down', () => {
  const b = { part: 'svc:redis', dependents: [dep({ how: 'queues its jobs on it', stops: [...many(36, 'route'), ...many(66, 'job')], unaffected: 262, unfollowed: 16 })], through, basis: '' }
  assert.equal(
    downSentence(b, name),
    'immich-server stops 36 routes and 66 background jobs that queue work on it, or are taken from its queue. 262 of its ways in reach nothing of it; 16 weren’t followed, so can’t be told.',
  )
})

test('a service whose callers cannot be tied: said, not guessed', () => {
  const b = { part: 'svc:ml', dependents: [dep({ how: 'waits for the answer', known: false })], through, basis: '' }
  assert.equal(downSentence(b, name), 'immich-server waits for its answer, but which of its ways in make the call isn’t recorded.')
  const two = { ...b, dependents: [dep({ id: 'app:web', how: 'waits for the answer', known: false }), dep({ id: 'app:mobile', how: 'waits for the answer', known: false }), dep({ id: 'actor:user', how: 'a person uses it' })] }
  assert.equal(downSentence(two, name), 'immich-web and immich_mobile wait for its answer, but which of their ways in make the call isn’t recorded.')
})

test('a part nothing uses, and a dependent none of whose ways in reach it', () => {
  assert.equal(downSentence({ part: 'svc:ml', dependents: [], through: [], basis: '' }, name), 'Nothing on this diagram uses immich-machine-learning, so nothing here breaks without it.')
  const none = { part: 'svc:database', dependents: [dep({ unaffected: 5 })], through: [], basis: '' }
  assert.equal(downSentence(none, name), 'immich-server keeps working: none of its ways in read or write its tables.')
})

test('what stops, by kind, in a fixed order', () => {
  assert.equal(stopsByKind([stop('job:a'), stop('route:b'), stop('command:c'), stop('page:d'), stop('route:e')]), '2 routes, 1 page, 1 command and 1 background job')
  assert.equal(stopsByKind([stop('route:x')]), '1 route')
})

test('the diagram: the part down, hatched; what depends on it tagged; the person not', () => {
  const b = { part: 'svc:database', dependents: [dep({}), dep({ id: 'actor:user', how: 'a person uses it' })], through, basis: '' }
  const h = downHighlight(b)
  assert.deepEqual(Object.fromEntries(h.tags!), { 'svc:database': 'Down', 'svc:immich-server': 'depends on it' })
  assert.deepEqual([...h.hatched!], ['svc:database'])
  assert.deepEqual([...h.lit].sort(), ['actor:user', 'app:web', 'svc:database', 'svc:immich-server'])
})

test('the list: what people wait on first, background jobs last', () => {
  assert.deepEqual(ordered([stop('job:a'), stop('route:z'), stop('command:c'), stop('route:b'), stop('page:p')]).map((e) => e.entry), ['route:b', 'route:z', 'page:p', 'command:c', 'job:a'])
})
