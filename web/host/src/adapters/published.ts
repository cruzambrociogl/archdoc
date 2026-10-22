// The published site, built by `archdoc export --site`: the same app reading the same answers from
// static files beside it (surface-spec §3). One version, nothing to save, and citations that open on
// the repository's host.

import type { Source } from '@archdoc/app'
import { staticName } from '@archdoc/app'
import { read } from './read'

const url = (path: string) => (path.startsWith('/api/') ? staticName(path) : path)

export const published: Source = {
  capabilities: { place: 'published', history: false, edit: false, runDetails: false, plans: false, links: 'host', ask: false },
  get: (path) => read(url(path)),
  url,
  act: () => Promise.reject(new Error('a published site is read-only')),
  // The repository host at the commit, when a remote is known; with none, a citation is plain text,
  // never a guessed link (surface-spec S-3).
  link(place, file, line) {
    if (!place.remote || !place.commit) return undefined
    return `${place.remote}/blob/${place.commit}/${file.replace(/^\.\//, '')}${line ? `#L${line}` : ''}`
  },
}
