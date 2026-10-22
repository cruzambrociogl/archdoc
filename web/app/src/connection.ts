// Whether the place the app reads from is answering (platform-surface §4.8, "archdoc or the network
// unreachable"). A read that cannot reach it at all — fetch's TypeError, not a server's answer —
// marks it down; any read that succeeds marks it back. While it is down the shell asks again every
// few seconds, and when it answers, every read that failed is asked again.

import { useEffect, useSyncExternalStore } from 'react'
import type { Source } from './source'

interface State {
  /** Since when nothing answers, and whether the browser says it is offline. */
  down?: { since: number; offline: boolean }
  /** Bumped to ask every failed read again: on reconnecting, and on Try again. */
  epoch: number
}

let state: State = { epoch: 0 }
const listeners = new Set<() => void>()
function set(next: State) {
  state = next
  listeners.forEach((l) => l())
}
const subscribe = (l: () => void) => (listeners.add(l), () => listeners.delete(l))

/** Did this failure mean nothing answered? fetch rejects with a TypeError only then. */
export const unanswered = (e: unknown): boolean => e instanceof TypeError

export function reached() {
  if (state.down) set({ epoch: state.epoch + 1 })
}

export function unreachable() {
  if (!state.down) set({ ...state, down: { since: Date.now(), offline: typeof navigator !== 'undefined' && !navigator.onLine } })
}

/** Asks every read that failed again. */
export function retryFailed() {
  set({ ...state, epoch: state.epoch + 1 })
}

export const useConnection = () => useSyncExternalStore(subscribe, () => state)

/** While nothing answers, asks again every few seconds and when the browser comes back online. */
export function useReconnect(source: Source) {
  const { down } = useConnection()
  useEffect(() => {
    const offline = () => (state.down ? set({ ...state, down: { ...state.down, offline: true } }) : unreachable())
    window.addEventListener('offline', offline)
    return () => window.removeEventListener('offline', offline)
  }, [])
  useEffect(() => {
    if (!down) return
    const probe = () =>
      source.get('/api/summary').then(reached, (e) => {
        if (!unanswered(e)) reached() // it answered, if not with this
      })
    const t = setInterval(probe, 5000)
    window.addEventListener('online', probe)
    return () => {
      clearInterval(t)
      window.removeEventListener('online', probe)
    }
  }, [down, source])
}
