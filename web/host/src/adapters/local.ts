// The app on this machine, served by `archdoc serve`: it reads the API, can save, and opens
// citations in the editor.

import type { Source } from '@archdoc/app'
import { read } from './read'

// Anything that runs or writes is a POST carrying this session's token, which the server checks
// together with the Origin (internal/serve/guard.go, surface-spec §11.5). The token is fetched once
// per page from /api/session, which answers same-origin pages only.
let session: Promise<string> | null = null

function token(): Promise<string> {
  session ??= read('/api/session')
    .then((r) => r.json())
    .then((s: { token: string }) => s.token)
    .catch((e) => {
      session = null
      throw e
    })
  return session
}

export const local: Source = {
  // What is designed and not built shows only while developing the app (platform-surface §4.4):
  // the product a user runs never carries placeholders.
  capabilities: { place: 'local', history: true, edit: true, runDetails: true, plans: import.meta.env.DEV, links: 'editor' },
  get: read,
  url: (path) => path,
  async act<T>(path: string, body?: unknown): Promise<T> {
    const r = await fetch(path, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Archdoc-Token': await token() },
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
