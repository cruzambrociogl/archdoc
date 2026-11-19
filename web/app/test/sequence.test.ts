// A scenario as a sequence and played on the parts: drawn from the flow it follows — lifelines for
// the parts its calls cross, one level down when it stays in one app, the answer dashed, a job's
// story all in the background — and the arrows its steps use, numbered.

import { test } from 'node:test'
import assert from 'node:assert/strict'
import type { Model, Scenario } from '../src/api.ts'
import { buildSequence, playedArrows, settle, words } from '../src/sequence.ts'

const model = {
  nodes: [
    { id: 'actor:user', name: 'User', kind: 'actor' },
    { id: 'app:api', name: 'api', kind: 'application' },
    { id: 'app:worker', name: 'worker', kind: 'application' },
    { id: 'svc:db', name: 'postgres', kind: 'datastore', technology: 'PostgreSQL 18' },
    { id: 'cmp:api/runs', name: 'internal/runs', kind: 'component', parent: 'app:api' },
    { id: 'cmp:api/store', name: 'internal/store', kind: 'component', parent: 'app:api' },
    { id: 'cmp:worker/build', name: 'internal/build', kind: 'component', parent: 'app:worker' },
    { id: 'cmp:worker/queries', name: 'internal/queries', kind: 'component', parent: 'app:worker' },
    { id: 'tbl:api/runs', name: 'runs', kind: 'table', parent: 'app:api' },
  ],
  edges: [
    { from: 'actor:user', to: 'app:api', label: 'uses' },
    { from: 'app:api', to: 'svc:db', label: 'connects to', traffic: true },
    { from: 'app:worker', to: 'svc:db', label: 'connects to', traffic: true },
  ],
  entries: [
    { id: 'route:api POST /runs', kind: 'http', method: 'POST', path: '/runs', handler: 'runs.Service.create', container: 'app:api', component: 'cmp:api/runs' },
    { id: 'job:worker build', kind: 'job', method: 'JOB', path: 'build', handler: 'build.Worker.Work', container: 'app:worker', component: 'cmp:worker/build' },
  ],
  flows: [
    {
      entry: 'route:api POST /runs',
      participants: [
        { id: 'internal/runs#Service', name: 'runs.Service', kind: 'class', component: 'cmp:api/runs' },
        { id: 'internal/store#Store', name: 'store.Store', kind: 'class', component: 'cmp:api/store' },
        { id: 'table:runs', name: 'runs', kind: 'table', element: 'tbl:api/runs' },
        { id: 'job:build', name: 'build', kind: 'job', element: 'job:worker build' },
      ],
      steps: [
        { from: 'internal/runs#Service', to: 'internal/store#Store', call: 'Add', depth: 0 },
        { from: 'internal/store#Store', to: 'table:runs', call: 'writes', depth: 1 },
        { from: 'internal/store#Store', to: 'job:build', call: 'queues', depth: 1 },
      ],
    },
    {
      entry: 'job:worker build',
      participants: [
        { id: 'internal/build#Worker', name: 'build.Worker', kind: 'class', component: 'cmp:worker/build' },
        { id: 'internal/queries#Queries', name: 'queries.Queries', kind: 'class', component: 'cmp:worker/queries' },
        { id: 'table:runs', name: 'runs', kind: 'table', element: 'tbl:api/runs' },
        { id: 'table:projects', name: 'projects', kind: 'table' },
      ],
      steps: [
        { from: 'internal/build#Worker', to: 'internal/queries#Queries', call: 'PrivateJob', depth: 0 },
        { from: 'internal/queries#Queries', to: 'table:runs', call: 'reads', depth: 1 },
        { from: 'internal/queries#Queries', to: 'table:projects', call: 'reads', depth: 1 },
        { from: 'internal/build#Worker', to: 'internal/build#Worker', call: 'fail', depth: 0 },
      ],
    },
  ],
} as unknown as Model

const upload: Scenario = {
  title: 'Asking for a run',
  flow: 'route:api POST /runs',
  mark: 'written-by-ai',
  steps: [
    { text: 'The API takes the request and stores the run.', from: 1, to: 2, crosses: ['cmp:api/runs', 'tbl:api/runs'] },
    { text: 'It queues the build for the worker.', from: 3, to: 3, background: true, crosses: ['cmp:api/store', 'job:worker build'] },
  ],
} as Scenario

const build: Scenario = {
  title: 'Building a run',
  flow: 'job:worker build',
  mark: 'written-by-ai',
  steps: [
    { text: 'The worker reads the run and its project.', from: 1, to: 3, crosses: ['cmp:worker/build', 'cmp:worker/queries', 'tbl:api/runs'] },
    { text: 'A failure is recorded.', from: 4, to: 4, crosses: ['cmp:worker/build'] },
  ],
} as Scenario

test('a story across apps: a lifeline per app, the calls between them, the answer dashed before what runs later', () => {
  const seq = buildSequence(upload, model)
  assert.equal(seq.from, 'flow')
  assert.equal(seq.inside, undefined)
  assert.deepEqual(seq.lanes.map((l) => `${l.kind} ${l.name}`), ['caller User', 'part api', 'part worker', 'store postgres'])
  assert.deepEqual(
    seq.messages.map((m) => `${m.step + 1} ${m.kind} ${m.from} > ${m.to}: ${m.label}${m.later ? ' (later)' : ''}`),
    [
      '1 call caller > app:api: POST /runs',
      '1 call app:api > svc:db: writes runs',
      '1 reply app:api > caller: the answer',
      '2 call app:api > app:worker: queues build (later)',
    ],
  )
})

test('a story inside one app: its parts, tables to its store, one message for the same reads, its own work a note — all in the background, a job’s', () => {
  const seq = buildSequence(build, model)
  assert.equal(seq.inside, 'worker')
  assert.deepEqual(seq.lanes.map((l) => l.name), ['internal/build', 'internal/queries', 'postgres'])
  assert.deepEqual(
    seq.messages.map((m) => `${m.step + 1} ${m.kind} ${m.label}${m.later ? ' (later)' : ''}`),
    ['1 call PrivateJob (later)', '1 call reads runs, projects (later)', '2 note A failure is recorded (later)'],
  )
})

test('a job’s story is in the background, every step; a route’s keeps what the explanation said', () => {
  assert.ok(settle(build, model).steps.every((s) => s.background))
  assert.equal(settle(upload, model), upload)
})

test('played on the parts: the arrows used carry their numbers, the current one its words', () => {
  const edges = model.edges ?? []
  const first = playedArrows(upload, 0, model, edges)
  assert.deepEqual([...first], [['app:api>svc:db', { label: '1 · The API takes the…', now: true }]])
  const later = playedArrows({ ...upload, steps: [upload.steps[0], { ...upload.steps[0], text: 'Then again.' }] }, 1, model, edges)
  assert.deepEqual(later.get('app:api>svc:db'), { label: '2 · Then again', now: true })
})

test('a few words of a sentence', () => {
  assert.equal(words('The worker finds runs that were skipped, by version.'), 'The worker finds runs that…')
  assert.equal(words('Short one.'), 'Short one')
})
