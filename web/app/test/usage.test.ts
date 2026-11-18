// Usage: the faces, their names and sentences, the filter, and what an entry sets off — on entries
// shaped like Immich's (its pages, routes, commands and jobs), and the cases it does not have.

import { test } from 'node:test'
import assert from 'node:assert/strict'
import { addressOf, countOf, faceMark, facesOf, matches, nameOf, setsOff, setsOffSentence, usedSentence } from '../src/screens/usage/faces.ts'

const at = { file: 'x.ts', line: 1 }
const entry = (kind: string, container: string, method: string, path: string, handler: string, summary?: string) =>
  ({ id: `${kind}:${container} ${method} ${path}`, kind, method, path, handler, container, summary, provenance: at }) as never

const entries = [
  entry('http', 'svc:immich-server', 'POST', '/api/assets', 'AssetMediaController.uploadAsset', 'Upload asset'),
  entry('http', 'svc:immich-server', 'GET', '/api/albums', 'AlbumController.getAllAlbums', 'List all albums'),
  entry('http', 'svc:immich-server', 'DELETE', '/api/albums/:id', 'AlbumController.deleteAlbum'),
  entry('http', 'svc:immich-machine-learning', 'POST', '/predict', 'predict'),
  entry('page', 'app:web', 'PAGE', '/photos', 'photos/+page.svelte'),
  entry('page', 'app:web', 'PAGE', '/albums', 'albums/+page.svelte'),
  entry('command', 'svc:immich-server', 'CMD', 'reset-admin-password', 'ResetAdminPasswordCommand.run', 'Reset the admin password'),
  entry('command', 'app:packages/cli', 'CMD', 'upload', 'upload'),
  entry('job', 'svc:immich-server', 'JOB', 'AssetDelete', 'AssetService.handleAssetDeletion'),
]
const names: Record<string, string> = {
  'svc:immich-server': 'immich-server', 'svc:immich-machine-learning': 'immich-machine-learning', 'app:web': 'immich-web', 'app:packages/cli': '@immich/cli',
}
const name = (id: string) => names[id] ?? id

test('the faces: screens, then APIs, then command lines; the part with most entries first; no background jobs', () => {
  const faces = facesOf(entries)
  assert.deepEqual(faces.map((f) => [f.key, f.entries.length]), [
    ['screens/app:web', 2],
    ['api/svc:immich-server', 3],
    ['api/svc:immich-machine-learning', 1],
    ['cli/app:packages/cli', 1],
    ['cli/svc:immich-server', 1],
  ])
  assert.ok(faces.every((f) => f.entries.every((e: { kind: string }) => e.kind !== 'job')))
})

test('a face’s entries in the order of their addresses', () => {
  const api = facesOf(entries).find((f) => f.key === 'api/svc:immich-server')!
  assert.deepEqual(api.entries.map(addressOf), ['DELETE /api/albums/:id', 'GET /api/albums', 'POST /api/assets'])
  assert.equal(countOf(api), '3 routes')
  assert.equal(countOf(facesOf(entries).find((f) => f.key === 'cli/app:packages/cli')!), '1 command')
})

test('an entry’s name is what its code says it does, else its address', () => {
  assert.equal(nameOf(entries[0]), 'Upload asset')
  assert.equal(nameOf(entries[2]), 'DELETE /api/albums/:id')
  assert.equal(nameOf(entries[4]), '/photos')
  assert.equal(nameOf(entries[6]), 'Reset the admin password')
  assert.equal(addressOf(entries[7]), 'upload')
})

test('the sentence naming the faces', () => {
  assert.equal(
    usedSentence('immich', facesOf(entries), name),
    'immich is used 5 ways: on the screens of immich-web, through the APIs of immich-server and immich-machine-learning, and from the command lines of @immich/cli and immich-server.',
  )
  assert.equal(usedSentence('api', facesOf([entries[0]]), name), 'api is used one way: through the API of immich-server.')
  assert.equal(usedSentence('lib', facesOf([entries[8]]), name), 'archdoc found no pages, routes or commands in lib’s code.')
})

test('each face says what it is made from, and that nothing is the AI’s', () => {
  const [screens, api, , cli] = facesOf(entries)
  assert.equal(faceMark(screens, []), 'From the code: each page’s path and the file that draws it. archdoc doesn’t read what a page shows or calls yet. Nothing here was written by AI.')
  // Where the engine followed a page, the mark does not say it can't.
  const followed = [{ entry: (screens.entries[0] as { id: string }).id, participants: [], steps: [] }] as never[]
  assert.equal(faceMark(screens, followed), 'From the code: each page’s path and the file that draws it. Nothing here was written by AI.')
  assert.match(faceMark(api, []), /^From the code: each route’s method, path and handler/)
  assert.match(faceMark(cli, []), /^From the code: each command’s name and handler/)
})

test('the filter matches a name, an address or a handler, in any case', () => {
  assert.ok(matches(entries[0], 'upload'))
  assert.ok(matches(entries[0], 'POST /api'))
  assert.ok(matches(entries[1], 'albumcontroller'))
  assert.ok(!matches(entries[1], 'assets'))
  assert.ok(matches(entries[1], '  '))
})

const flow = (steps: [string, string, string][], cut = false) =>
  ({
    entry: 'e',
    cut,
    participants: [
      { id: 'svc', name: 'AssetService', kind: 'class' },
      { id: 'table:asset', name: 'asset', kind: 'table' },
      { id: 'table:asset_exif', name: 'asset_exif', kind: 'table' },
      { id: 'table:user', name: 'user', kind: 'table' },
      { id: 'job:SidecarWrite', name: 'SidecarWrite', kind: 'job' },
      { id: 'unresolved', name: 'an address computed at run time', kind: 'unresolved' },
    ],
    steps: steps.map(([from, to, call]) => ({ from, to, call, depth: 1, provenance: at })),
  }) as never

test('what an entry sets off: tables by what it does to them, jobs queued, calls it cannot follow', () => {
  const s = setsOff(flow([['svc', 'table:asset', 'reads'], ['svc', 'table:user', 'reads'], ['svc', 'table:asset', 'reads'], ['svc', 'table:asset_exif', 'writes'], ['svc', 'table:user', 'updates'], ['svc', 'table:asset', 'deletes'], ['svc', 'job:SidecarWrite', 'queues'], ['svc', 'unresolved', 'fetch']]))!
  assert.deepEqual(s, { reads: ['asset', 'user'], writes: ['asset_exif'], updates: ['user'], deletes: ['asset'], queues: ['SidecarWrite'], loose: 1, cut: false })
  assert.equal(setsOffSentence(s), 'Reads asset and user; writes asset_exif; updates user; deletes asset; queues SidecarWrite; makes a call to an address set at run time.')
})

test('what it sets off, when nothing was followed, nothing was found, or the following stopped', () => {
  assert.equal(setsOffSentence(setsOff(undefined)), 'archdoc didn’t follow what it does.')
  assert.equal(setsOffSentence(setsOff(flow([]))), 'No tables, jobs or calls archdoc could follow.')
  assert.equal(setsOffSentence(setsOff(flow([], true))), 'Nothing archdoc could follow before it stopped.')
  assert.equal(setsOffSentence(setsOff(flow([['svc', 'table:asset', 'reads']], true))), 'Reads asset; and more, past where archdoc stopped following.')
})
