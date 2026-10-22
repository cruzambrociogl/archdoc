// The address's path, for the levels above a repository — Home, a system — which live in the path
// while a repository's own pages live after `#` (route.ts). One store, so every level reads the same
// path and moves it the same way.

import { useSyncExternalStore } from 'react'

let current = location.pathname
const listeners = new Set<() => void>()
const emit = () => listeners.forEach((l) => l())
window.addEventListener('popstate', () => {
  current = location.pathname
  emit()
})

/**
 * Moves to a path. Entering a repository, the address changes first, so its router starts from it.
 * Leaving one — a page after `#` is showing — the page goes first and the address after: the
 * repository's router, still listening while it is on screen, would otherwise take the new address
 * for one of its own and add its page to it.
 */
export function goTo(path: string) {
  const leaving = location.hash !== ''
  if (!leaving) history.pushState(null, '', path)
  current = new URL(path, location.href).pathname // a repository's page may follow, after #
  emit()
  if (leaving) setTimeout(() => history.pushState(null, '', path), 0)
}

/** Replaces the address — a level with one place goes straight to it. */
export function replaceWith(path: string) {
  history.replaceState(null, '', path)
  current = path
  emit()
}

export const usePath = () =>
  useSyncExternalStore(
    (l) => (listeners.add(l), () => listeners.delete(l)),
    () => current,
  )
