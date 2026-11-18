// The addresses: every view has one, and reading it back gives the same view.

import { test } from 'node:test'
import assert from 'node:assert/strict'
import { hrefOf, parseSearch, routeAt } from '../src/route.ts'
import type { Route } from '../src/route.ts'

/** What the router hands routeAt: the path, and the search already parsed. */
const read = (href: string) => {
  const [path, search = ''] = href.split('?')
  return routeAt(path, parseSearch(search))
}

const views: [Route, string][] = [
  [{ screen: 'home' }, '/home'],
  [{ screen: 'architecture', level: 'context' }, '/architecture/context'],
  [{ screen: 'architecture', level: 'deployment' }, '/architecture/deployment'],
  [{ screen: 'architecture', level: 'container', look: 'classic' }, '/architecture/containers?look=classic'],
  [{ screen: 'architecture', level: 'container', focus: 'svc:immich-server' }, '/architecture/containers?part=svc%3Aimmich-server'],
  [{ screen: 'architecture', level: 'component:svc:immich-server', all: '1' }, '/architecture/components/svc%3Aimmich-server?all=1'],
  [{ screen: 'architecture', level: 'structure:svc:immich-server' }, '/architecture/components/svc%3Aimmich-server?by=folder'],
  [{ screen: 'part', focus: 'svc:immich-machine-learning', question: 'touches' }, '/architecture/part/svc%3Aimmich-machine-learning/touches'],
  [{ screen: 'part', focus: 'app:packages/cli', question: 'depends' }, '/architecture/part/app%3Apackages%2Fcli/depends'],
  [{ screen: 'part', focus: 'svc:redis', question: 'depends', q: 'immich' }, '/architecture/part/svc%3Aredis/depends?q=immich'],
  [{ screen: 'data', level: 'data:svc:immich-server' }, '/data/data/svc%3Aimmich-server'],
  [{ screen: 'data', level: 'dataflow' }, '/data/dataflow'],
  [{ screen: 'features' }, '/features'],
  [{ screen: 'usage', focus: 'route:GET /api/albums' }, '/usage?entry=route%3AGET+%2Fapi%2Falbums'],
  [{ screen: 'usage', face: 'api/svc:immich-server', focus: 'route:GET /api/albums' }, '/usage/api/svc%3Aimmich-server?entry=route%3AGET+%2Fapi%2Falbums'],
  [{ screen: 'usage', face: 'cli/app:packages/cli', q: 'up' }, '/usage/cli/app%3Apackages%2Fcli?q=up'],
  [{ screen: 'entries', focus: 'job:immich-server AssetDelete' }, '/reference/entry-points?entry=job%3Aimmich-server+AssetDelete'],
  [{ screen: 'entries', q: 'album' }, '/reference/entry-points?q=album'],
  [{ screen: 'changes', v: 3, from: 1 }, '/changes?v=3&from=1'],
  [{ screen: 'change', focus: 'route:server POST /api/ocr' }, '/changes/latest/route%3Aserver%20POST%20%2Fapi%2Focr'],
  [{ screen: 'change', v: 7, focus: 'w0' }, '/changes/7/w0'],
  [{ screen: 'change', focus: 'cmp:server/ocr', from: 3 }, '/changes/latest/cmp%3Aserver%2Focr?from=3'],
  [{ screen: 'plan' }, '/plan'],
  [{ screen: 'reference' }, '/reference'],
  [{ screen: 'component', focus: 'cmp:immich-server/asset' }, '/reference/component/cmp%3Aimmich-server%2Fasset'],
  [{ screen: 'deps', q: 'nest' }, '/reference/dependencies?q=nest'],
  [{ screen: 'docs', doc: 'docs/architecture/01.generated.md' }, '/reference/documentation?doc=docs%2Farchitecture%2F01.generated.md'],
  [{ screen: 'about' }, '/about/what-was-read'],
  [{ screen: 'about', all: '1' }, '/about/what-was-read?all=1'],
  [{ screen: 'written' }, '/about/what-the-ai-wrote'],
  [{ screen: 'report' }, '/about/report'],
  [{ screen: 'runs', run: 4 }, '/about/what-left?run=4'],
  [{ screen: 'settings' }, '/settings'],
]

for (const [route, href] of views) {
  test(`${href} is the address of its view, and opens it again`, () => {
    assert.equal(hrefOf(route), href)
    assert.deepEqual(read(href), route)
  })
}

test('a router that hands over the path already decoded still finds the part', () => {
  assert.deepEqual(routeAt('/architecture/part/app:packages/cli/depends', {}), { screen: 'part', focus: 'app:packages/cli', question: 'depends' })
  assert.deepEqual(routeAt('/reference/component/cmp:immich-server/asset', {}), { screen: 'component', focus: 'cmp:immich-server/asset' })
})

test('a face whose part holds a slash, from a router that decodes the path', () => {
  assert.deepEqual(routeAt('/usage/cli/app:packages/cli', {}), { screen: 'usage', face: 'cli/app:packages/cli' })
})

test('the look belongs to Architecture: a link elsewhere does not carry it', () => {
  assert.equal(hrefOf({ screen: 'data', level: 'dataflow', look: 'classic' }), '/data/dataflow')
  assert.deepEqual(routeAt('/data/dataflow', parseSearch('look=classic')), { screen: 'data', level: 'dataflow' })
})

test('the bare address is Home', () => {
  assert.deepEqual(routeAt('/', {}), { screen: 'home' })
})

test('an address no page lives at is none', () => {
  for (const path of ['/map', '/overview', '/does', '/architecture/elsewhere', '/architecture/part/svc%3Aredis', '/architecture/part/svc%3Aredis/breaks', '/data/nothing', '/not-built-yet', '/usage/jobs/svc%3Ax', '/usage/api', '/changes/x/w0', '/changes/0/w0', '/architecture/deployment/x']) {
    assert.equal(routeAt(path, {}), undefined, path)
  }
})
