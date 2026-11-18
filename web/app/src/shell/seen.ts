// "Since you last looked" (platform-surface §17.3, Home on a return visit): which version this
// browser last showed for a repository. Read once per page load, before it is moved to the latest,
// so every screen of this visit — Home, the menu, Changes — agrees on what is new. Storage may be
// blocked or empty; then there is no "since", only the versions.

import { useEffect } from 'react'

const atLoad = new Map<string, number | undefined>()
const key = (root: string) => `archdoc.seen.${root}`

/** The version last seen before this visit, read once and kept for the rest of it. */
export function seenAtLoad(root: string): number | undefined {
  if (!atLoad.has(root)) {
    let v: number | undefined
    try {
      const raw = localStorage.getItem(key(root))
      v = raw === null ? undefined : Number(raw)
      if (!Number.isInteger(v)) v = undefined
    } catch {
      v = undefined
    }
    atLoad.set(root, v)
  }
  return atLoad.get(root)
}

/** The version last seen before this visit; this visit is then remembered as having seen the latest. */
export function useSeen(root: string, latest: number | undefined, remember: boolean): number | undefined {
  const seen = remember ? seenAtLoad(root) : undefined
  useEffect(() => {
    if (!remember || latest === undefined) return
    try {
      localStorage.setItem(key(root), String(latest))
    } catch {
      // Blocked: nothing remembered.
    }
  }, [root, latest, remember])
  return seen
}
