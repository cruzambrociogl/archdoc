import { useEffect, useState } from 'react'
import type { UpdateEnd } from '../api'
import type { Source } from '../source'
import { useCapabilities, useSource } from '../source'

/** Updates of this repository on this machine: one running now — from the terminal, or from here — and the last one the app ran. */
export interface Updating {
  running: boolean
  /** The one running was started from the app; ai, with AI. */
  mine?: boolean
  ai?: boolean
  by?: string
  since?: string
  doing?: string
  found?: string[]
  last?: UpdateEnd
}

// One watch per repository, however many parts of the page follow it: the top bar's pill and the
// notice band ask once between them.
interface Watch {
  state?: Updating
  listeners: Set<(u: Updating) => void>
  timer?: number
  look: () => Promise<void>
}
const watches = new Map<Source, Watch>()

function watch(source: Source): Watch {
  const found = watches.get(source)
  if (found) return found
  const w: Watch = {
    listeners: new Set(),
    look: async () => {
      if (document.hidden) return
      try {
        const u: Updating = await (await source.get('/api/updating')).json()
        w.state = u
        w.listeners.forEach((l) => l(u))
      } catch {
        /* the connection's notice says so */
      }
    },
  }
  watches.set(source, w)
  return w
}

/** Updates of this repository, asked every few seconds while the page is visible — on this machine only. */
export function useUpdating(): Updating | undefined {
  const source = useSource()
  const can = useCapabilities()
  const [u, setU] = useState<Updating | undefined>(() => watches.get(source)?.state)
  useEffect(() => {
    if (can.place !== 'local') return
    const w = watch(source)
    w.listeners.add(setU)
    if (w.listeners.size === 1) {
      void w.look()
      w.timer = window.setInterval(() => void w.look(), 3000)
    } else if (w.state) setU(w.state)
    return () => {
      w.listeners.delete(setU)
      if (!w.listeners.size) {
        clearInterval(w.timer)
        watches.delete(source)
      }
    }
  }, [can.place, source])
  return u
}

/** Asks now rather than at the next tick: an update was just started. */
export function lookNow(source: Source) {
  void watches.get(source)?.look()
}

/** When this page was loaded: an update that ended before it is already on it. */
export const loadedAt = Date.now()
