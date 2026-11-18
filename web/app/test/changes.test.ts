// Changes: where each change landed, lifted to the boxes a diagram draws; and Home's return visit,
// its sentence and the plan's verdict in one line.

import { test } from 'node:test'
import assert from 'node:assert/strict'
import { landed, tell } from '../src/story.ts'
import { planLine, sinceSentence } from '../src/screens/home/since.ts'

const at = (file: string, line: number) => ({ file, line })
const node = (id: string, kind: string, name: string, parent?: string, extra: object = {}) => ({ id, kind, name, parent, evidence: 'declared', provenance: at('compose.yml', 1), ...extra })
const before = {
  model: {
    nodes: [
      node('svc:server', 'application', 'immich-server'),
      node('svc:redis', 'datastore', 'redis', undefined, { technology: 'Redis 7' }),
      node('svc:ml', 'application', 'machine-learning'),
      node('cmp:server/asset', 'component', 'asset', 'svc:server'),
      node('tbl:server/asset', 'table', 'asset', 'svc:server'),
    ],
    edges: [],
  },
} as never
const after = {
  model: {
    nodes: [
      node('svc:server', 'application', 'immich-server'),
      node('svc:redis', 'datastore', 'redis', undefined, { technology: 'Valkey 9' }),
      node('svc:ml', 'application', 'machine-learning'),
      node('cmp:server/asset', 'component', 'asset', 'svc:server'),
      node('cmp:server/ocr', 'component', 'ocr', 'svc:server'),
      node('tbl:server/asset', 'table', 'asset', 'svc:server'),
      node('tbl:server/ocr', 'table', 'ocr', 'svc:server'),
      node('ext:maps', 'external', 'tiles.immich.cloud'),
    ],
    edges: [],
  },
} as never
const diff = {
  from: 1, to: 2, structural: true, empty: false,
  diff: {
    added_nodes: [node('cmp:server/ocr', 'component', 'ocr', 'svc:server'), node('tbl:server/ocr', 'table', 'ocr', 'svc:server'), node('ext:maps', 'external', 'tiles.immich.cloud')],
    removed_nodes: [],
    added_edges: [{ from: 'svc:server', to: 'svc:ml', label: 'calls', provenance: [at('compose.yml', 9)] }],
    removed_edges: [],
    changed: [
      { element: 'svc:redis', field: 'technology', before: 'Redis 7', after: 'Valkey 9' },
      { element: 'tbl:server/asset', field: 'column ocr_text', before: '', after: 'text' },
      { element: 'svc:server', field: 'description', before: 'a', after: 'b' },
    ],
    added_entries: [{ id: 'route:server POST /api/ocr', kind: 'http', method: 'POST', path: '/api/ocr', handler: 'OcrController.run', container: 'svc:server', provenance: at('ocr.controller.ts', 12) }],
    removed_entries: [{ id: 'job:server Old', kind: 'job', method: 'JOB', path: 'Old', handler: 'x', container: 'svc:server', provenance: at('old.ts', 3) }],
  },
} as never

const { groups } = tell(diff, before, after)
const item = (key: string) => groups.flatMap((g) => g.items).find((i) => i.key === key)!
const drawn = new Set(['svc:server', 'svc:redis', 'svc:ml', 'ext:maps', 'actor:user'])
const parents = new Map<string, string>([['cmp:server/asset', 'svc:server'], ['cmp:server/ocr', 'svc:server'], ['tbl:server/asset', 'svc:server'], ['tbl:server/ocr', 'svc:server']])
const where = (key: string) => {
  const l = landed(item(key).on ?? [], (id) => parents.get(id), drawn)
  return { lit: [...l.lit].sort(), tags: Object.fromEntries(l.tags) }
}

test('a new route lands on the part it is in, tagged', () => {
  assert.deepEqual(where('route:server POST /api/ocr'), { lit: ['svc:server'], tags: { 'svc:server': 'New route' } })
})

test('a new component lands on the part it is inside', () => {
  assert.deepEqual(where('cmp:server/ocr'), { lit: ['svc:server'], tags: { 'svc:server': 'New part inside' } })
})

test('a new outside service lands on itself', () => {
  assert.deepEqual(where('ext:maps'), { lit: ['ext:maps'], tags: { 'ext:maps': 'New' } })
})

test('a new technology says what it is now', () => {
  assert.deepEqual(where('v0'), { lit: ['svc:redis'], tags: { 'svc:redis': 'Now Valkey 9' } })
})

test('a new table and a changed column land on the part whose code declares them', () => {
  assert.deepEqual(where('tbl:server/ocr'), { lit: ['svc:server'], tags: { 'svc:server': 'New table' } })
  assert.deepEqual(where('c0'), { lit: ['svc:server'], tags: { 'svc:server': 'Columns changed' } })
})

test('a new connection lights both ends, the tag on the one that now calls', () => {
  assert.deepEqual(where('w0'), { lit: ['svc:ml', 'svc:server'], tags: { 'svc:server': 'New connection' } })
})

test('a route gone lands on its part', () => {
  assert.deepEqual(where('job:server Old'), { lit: ['svc:server'], tags: { 'svc:server': 'Job gone' } })
})

test('wording only does not land anywhere', () => {
  const wording = groups.find((g) => g.title.startsWith('Wording'))!
  assert.ok(wording.items.every((i) => !i.on))
})

test('two things on one box say both, once each; what is drawn nowhere is left out', () => {
  const l = landed([{ id: 'svc:server', tag: 'New route' }, { id: 'cmp:server/ocr', tag: 'New part inside' }, { id: 'svc:server', tag: 'New route' }, { id: 'svc:gone', tag: 'Gone' }], (id) => parents.get(id), drawn)
  assert.deepEqual([...l.lit], ['svc:server'])
  assert.equal(l.tags.get('svc:server'), 'New route · New part inside')
})

test('the return visit’s sentence', () => {
  assert.equal(sinceSentence(2, true, ['added 1 part and 1 way in', 'changed 1 detail']), '2 new versions since you last looked changed the architecture: added 1 part and 1 way in; changed 1 detail.')
  assert.equal(sinceSentence(1, false, []), '1 new version since you last looked; it didn’t change the architecture.')
  assert.equal(sinceSentence(3, false, []), '3 new versions since you last looked; none changed the architecture.')
})

const verdict = (status: string) => ({ item: { id: status, text: '', provenance: at('plan.md', 1) }, status, claim: { text: '', facts: [], cites: [] } })
test('the plan’s verdict in one line, never a score', () => {
  assert.equal(planLine({ verdicts: [verdict('built'), verdict('built'), verdict('missing')], extras: null, facts: 0, additions: 0 } as never), 'The plan asked for 3 things. 2 look built; 1 wasn’t found.')
  assert.equal(planLine({ verdicts: [verdict('unclear')], extras: null, facts: 0, additions: 0 } as never), 'The plan asked for one thing. 1 is unclear.')
  assert.equal(planLine({ verdicts: [verdict('missing'), verdict('missing'), verdict('built'), verdict('unclear')], extras: null, facts: 0, additions: 0 } as never), 'The plan asked for 4 things. 1 looks built; 2 weren’t found; 1 is unclear.')
})
