// Where the exported data answers an API path: the published site reads it beside the page, the
// hosted website through its API. One mirror of serve.StaticName in Go — change one, change both.

/** The file the exported data holds an API path's answer in. Mirrors serve.StaticName in Go. */
export function staticName(apiPath: string): string {
  const [p, query = ''] = apiPath.replace(/#.*$/, '').replace(/^\/api\//, '').split('?')
  if (p.startsWith('docs/')) return `data/${p}`
  const ext = p === 'svg' ? '.svg' : '.json'
  const q = new URLSearchParams(query)
  const keys = [...new Set([...q.keys()])].sort()
  // A view name may hold an element ID ("component:app:packages/cli"): no slash or colon in a file name.
  const safe = (v: string | null) => (v ?? '').replace(/\//g, '_').replace(/:/g, '~')
  const tail = keys.length ? '@' + keys.map((k) => `${k}=${safe(q.get(k))}`).join(',') : ''
  return `data/${p.replace(/\//g, '_')}${tail}${ext}`
}

