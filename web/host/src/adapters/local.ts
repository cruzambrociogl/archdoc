// The app on this machine, served by `archdoc serve`: it reads the API, can save, and opens
// citations in the editor.

import type { Source } from '@archdoc/app'
import { read } from './read'

// Anything that runs or writes is a POST carrying this session's token, which the server checks
// together with the Origin (internal/serve/guard.go, surface-spec §11.5). The token is fetched once
// per page from /api/session, which answers same-origin pages only. In a system, each repository's
// server has its own session, under its own prefix.
const sessions = new Map<string, Promise<string>>()

function token(prefix: string): Promise<string> {
  let session = sessions.get(prefix)
  if (!session) {
    session = read(`${prefix}/api/session`)
      .then((r) => r.json())
      .then((s: { token: string }) => s.token)
      .catch((e) => {
        sessions.delete(prefix)
        throw e
      })
    sessions.set(prefix, session)
  }
  return session
}

/** The app for one repository served by `archdoc serve`, its API under prefix — "" alone, "/repos/{name}" in a system. */
export function localAt(prefix: string): Source {
  return {
    // What is designed and not built shows only while developing the app (platform-surface §4.4):
    // the product a user runs never carries placeholders.
    capabilities: { place: 'local', history: true, edit: true, runDetails: true, plans: import.meta.env.DEV, links: 'editor', ask: true },
    get: (path) => read(prefix + path),
    url: (path) => prefix + path,
    async act<T>(path: string, body?: unknown): Promise<T> {
      const r = await fetch(prefix + path, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-Archdoc-Token': await token(prefix) },
        body: body === undefined ? undefined : JSON.stringify(body),
      })
      const text = await r.text()
      let data: unknown = undefined
      try {
        data = text ? JSON.parse(text) : undefined
      } catch {
        /* not JSON */
      }
      if (!r.ok) throw new Error((data as { error?: string } | undefined)?.error ?? r.statusText)
      return data as T
    },
    link(place, file, line) {
      const path = file.startsWith('/') ? file : `${place.root}/${file}`
      return `vscode://file${path}${line ? `:${line}` : ''}`
    },
  }
}

export const local = localAt('')

/** A folder of repositories served as one system, at prefix: its map, and each repository under its name. */
function systemAt(prefix: string): Source {
  return {
    ...localAt(prefix),
    system: {
      get: () => read(`${prefix}/api/system`).then((r) => r.json()),
      open: (repo) => localAt(`${prefix}/repos/${encodeURIComponent(repo)}`),
    },
  }
}

export const system = systemAt('')

/** `archdoc home`: every place this machine remembers, each under /at/{id}. */
export const home: Source = {
  ...local,
  home: {
    get: () => read('/api/home').then((r) => r.json()),
    open: (place) => (place.kind === 'system' ? systemAt : localAt)(`/at/${place.id}`),
    opened: (id) => local.act('/api/home/open', { id }),
  },
}
