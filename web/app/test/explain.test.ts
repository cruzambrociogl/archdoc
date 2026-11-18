// The system explained, as the app reads it: citations lead to the facts they name, features and
// scenarios light the boxes their facts sit in, and nothing is invented for an ID that names nothing.

import { test } from 'node:test'
import assert from 'node:assert/strict'
import type { Interpretation, Model } from '../src/api.ts'
import { areaOf, boxesOf, boxesOfAll, citeTarget, featureAt, featureSlugs, featuresThrough, scenarioAt, scenarioReach, scenariosThrough, slug, stepsLine, tablesReached } from '../src/explain.ts'

const model = {
  nodes: [
    { id: 'actor:user', name: 'User', kind: 'actor' },
    { id: 'svc:server', name: 'server', kind: 'application' },
    { id: 'app:web', name: 'web', kind: 'application' },
    { id: 'svc:db', name: 'db', kind: 'datastore' },
    { id: 'cmp:server/asset', name: 'asset', kind: 'component', parent: 'svc:server' },
    { id: 'cmp:server/face', name: 'face', kind: 'component', parent: 'svc:server' },
    { id: 'tbl:asset', name: 'asset', kind: 'table', parent: 'svc:db' },
  ],
  edges: [{ from: 'app:web', to: 'svc:server', label: 'calls the API of' }],
  entries: [
    { id: 'route:server POST /api/assets', kind: 'http', method: 'POST', path: '/api/assets', handler: 'A.upload', container: 'svc:server', component: 'cmp:server/asset' },
    { id: 'job:server FaceDetect', kind: 'job', method: '', path: 'FaceDetect', handler: 'F.detect', container: 'svc:server', component: 'cmp:server/face' },
  ],
} as unknown as Model

const it = {
  thesis: { text: 'A photo server.', cites: ['svc:server'], mark: 'written-by-ai' },
  features: [
    { name: 'Photo library', says: 'Keeping photos.', members: ['route:server POST /api/assets'], parts: ['cmp:server/asset'], mark: 'written-by-ai' },
    { name: 'Faces', says: 'Finding people.', members: ['job:server FaceDetect'], parts: ['cmp:server/face'], mark: 'written-by-ai' },
    { name: 'Faces', says: 'Again, by name.', members: [], mark: 'proven' },
  ],
  concepts: [{ name: 'Photos', says: 'What is kept.', members: ['tbl:asset'], mark: 'written-by-ai' }],
  areas: [{ name: 'Library', says: 'Photos.', members: ['cmp:server/asset', 'cmp:server/face'], everywhere: true, mark: 'written-by-ai' }],
  areas_of: 'svc:server',
  scenarios: [
    {
      title: 'Uploading a photo',
      flow: 'route:server POST /api/assets',
      mark: 'written-by-ai',
      steps: [
        { text: 'The web app sends it.', from: 1, to: 2, crosses: ['cmp:server/asset'] },
        { text: 'It is written.', from: 3, to: 3, crosses: ['tbl:asset'] },
        { text: 'Faces, later.', from: 4, to: 4, background: true, crosses: ['job:server FaceDetect'] },
      ],
    },
  ],
  review: [],
  calls: [],
  cost: 0,
  cost_known: true,
} as unknown as Interpretation

test('a name is an address, and two features of one name get two', () => {
  assert.equal(slug('Face and people recognition'), 'face-and-people-recognition')
  assert.equal(slug('Sign-in & accounts!'), 'sign-in-accounts')
  assert.deepEqual(featureSlugs(it), ['photo-library', 'faces', 'faces-2'])
  assert.equal(featureAt(it, 'faces-2')?.index, 2)
  assert.equal(featureAt(it, 'nothing'), undefined)
  assert.equal(scenarioAt(it, 'uploading-a-photo')?.scenario.title, 'Uploading a photo')
})

test('a fact lights the box it sits in on Apps and services', () => {
  assert.deepEqual(boxesOf('cmp:server/asset', model), ['svc:server'])
  assert.deepEqual(boxesOf('table:asset', model), ['svc:db'])
  assert.deepEqual(boxesOf('job:server FaceDetect', model), ['svc:server'])
  assert.deepEqual(boxesOf('app:web>svc:server', model), ['app:web', 'svc:server'])
  assert.deepEqual(boxesOf('feature:1', model), [], 'a group sits in no box')
  assert.deepEqual(boxesOfAll(['cmp:server/asset', 'tbl:asset', 'cmp:server/face'], model), ['svc:server', 'svc:db'])
})

test('a citation opens what it names, and an unknown one opens nothing', () => {
  assert.deepEqual(citeTarget('feature:2', model, it), { label: 'Faces', to: { screen: 'feature', focus: 'faces' } })
  assert.equal(citeTarget('route:server POST /api/assets', model, it)?.label, 'POST /api/assets')
  assert.deepEqual(citeTarget('cmp:server/asset', model, it)?.to, { screen: 'component', focus: 'cmp:server/asset' })
  assert.deepEqual(citeTarget('svc:db', model, it)?.to, { screen: 'architecture', level: 'container', focus: 'svc:db' })
  assert.equal(citeTarget('app:web>svc:server', model, it)?.label, 'web → server')
  assert.equal(citeTarget('svc:nowhere', model, it), undefined)
  assert.equal(citeTarget('feature:9', model, it), undefined)
})

test('a scenario says how its steps divide, and what it has reached by each one', () => {
  const s = it.scenarios![0]
  assert.equal(stepsLine(s), '3 steps · 2 while you wait, 1 in the background')
  assert.deepEqual(scenarioReach(s, 0, model), { reached: ['svc:server'], current: ['svc:server'] })
  assert.deepEqual(scenarioReach(s, 1, model), { reached: ['svc:server', 'svc:db'], current: ['svc:db'] })
})

test('a part names the features and scenarios through it, and areas group their components', () => {
  assert.deepEqual(featuresThrough(it, 'svc:server', model).map((f) => f.name), ['Photo library', 'Faces'])
  assert.deepEqual(scenariosThrough(it, 'svc:db', model).map((s) => s.title), ['Uploading a photo'])
  assert.deepEqual(scenariosThrough(it, 'app:web', model), [])
  assert.equal(areaOf(it).get('cmp:server/face'), 'Library')
})

test('a table lights the one database it lives in, a job the one queue store — the engine’s rules, never a guess', () => {
  const m = {
    nodes: [
      { id: 'svc:server', name: 'server', kind: 'application' },
      { id: 'svc:db', name: 'db', kind: 'datastore', technology: 'PostgreSQL 14' },
      { id: 'svc:redis', name: 'redis', kind: 'datastore', technology: 'Valkey 9' },
      { id: 'tbl:server/asset', name: 'asset', kind: 'table', parent: 'svc:server' },
    ],
    edges: [
      { from: 'svc:server', to: 'svc:db', label: 'connects to', traffic: true },
      { from: 'svc:server', to: 'svc:redis', label: 'connects to', traffic: true },
    ],
    entries: [{ id: 'job:server Thumbnail', kind: 'job', method: '', path: 'Thumbnail', handler: 'T.run', container: 'svc:server' }],
    flows: [{ entry: 'job:server Thumbnail', participants: [{ id: 't', name: 'asset', kind: 'table', element: 'tbl:server/asset' }], steps: [] }],
  } as unknown as Model
  assert.deepEqual(boxesOf('tbl:server/asset', m), ['svc:server', 'svc:db'])
  assert.deepEqual(boxesOf('job:server Thumbnail', m), ['svc:server', 'svc:redis'])
  assert.deepEqual(tablesReached(['job:server Thumbnail'], m), ['tbl:server/asset'])
  // Two relational stores: where the tables live is not known, so only the container is lit.
  const two = { ...m, nodes: [...m.nodes!, { id: 'svc:db2', name: 'db2', kind: 'datastore', technology: 'MySQL' }], edges: [...m.edges!, { from: 'svc:server', to: 'svc:db2', label: 'connects to', traffic: true }] } as unknown as Model
  assert.deepEqual(boxesOf('tbl:server/asset', two), ['svc:server'])
})
