import type { ReactNode } from 'react'
import { createContext, useContext, useEffect, useState } from 'react'
import type { Summary, Version } from '../api'
import { notAnswering } from '../api'
import { retryFailed, useConnection } from '../connection'
import { useCapabilities, useSource } from '../source'

/**
 * A notice above the page (platform-surface §4.8; App.dc.html): what happened, in one sentence;
 * what is still true; what to do, if anything. The page under it never changes on its own — a
 * notice says that something did, and the reader chooses.
 */
export interface Notice {
  id: string
  glyph: 'warn' | 'info' | 'changed'
  what: ReactNode
  still?: ReactNode
  todo?: ReactNode
  acts?: { label: string; on: () => void; primary?: boolean }[]
  /** Folded to its first sentence, with "What still works" to open the rest. */
  fold?: boolean
}

const NoticeContext = createContext<Notice[]>([])

/** Notices a host gives the app about what it shows — the website's, about the run it opened. */
export const NoticeProvider = NoticeContext.Provider

const glyphs = { warn: '!', info: 'i', changed: 'Δ' }

const time = (ms: number) => new Date(ms).toLocaleTimeString(undefined, { hour: 'numeric', minute: '2-digit' })
const day = (iso: string) => new Date(iso).toLocaleDateString(undefined, { day: 'numeric', month: 'short' })

/** The one notice that matters most now: the connection, then a newer version, then the host's. */
export function NoticeBand({ summary }: { summary: Summary }) {
  const can = useCapabilities()
  const { down } = useConnection()
  const given = useContext(NoticeContext)
  const busy = useUpdating()
  const newer = useNewer(summary, busy?.since)
  const [dismissed, setDismissed] = useState<string[]>([])
  const dismiss = (id: string) => () => setDismissed((d) => [...d, id])

  const notices: Notice[] = []
  if (down) {
    const since = time(down.since)
    const still = 'This page keeps working with what it already loaded. Pages not opened yet wait for it.'
    notices.push(
      down.offline
        ? { id: 'offline', glyph: 'warn', what: `You’re offline since ${since}.`, still, todo: 'It reconnects on its own when the network is back.', acts: [{ label: 'Try now', on: retryFailed, primary: true }], fold: true }
        : {
            id: 'down',
            glyph: 'warn',
            what: `${can.place === 'published' ? 'This site isn’t answering' : notAnswering[can.place]} since ${since}.`,
            still: can.place === 'hosted' ? `The problem is on archdoc’s side, not yours. ${still}` : still,
            todo:
              can.place === 'local'
                ? 'Start it again in a terminal — archdoc serve, in the repository’s folder. This page reconnects on its own.'
                : 'Nothing to do here; it tries again on its own.',
            acts: [{ label: 'Try now', on: retryFailed, primary: true }],
            fold: true,
          },
    )
  }
  if (busy) {
    // Two updates at once (surface-spec §11.5): this page waits for the one running, and says so.
    notices.push({
      id: `updating-${busy.since}`,
      glyph: 'info',
      what: `An update ${busy.by === 'the app' ? 'started here' : 'started in the terminal'} is running: ${busy.by}, since ${time(Date.parse(busy.since))}.`,
      still: busy.doing ? `Now ${busy.doing}${busy.found?.length ? ` — ${busy.found.join(', ')} so far` : ''}.` : undefined,
      todo: 'These docs stay as they are until it ends; then this page says what changed.',
      acts: [],
    })
  }
  if (newer && !dismissed.includes(`newer-${newer.id}`)) {
    notices.push({
      id: `newer-${newer.id}`,
      glyph: 'changed',
      what: (
        <>
          The architecture changed: archdoc recorded a new version, from commit <span className="mono">{newer.commit.slice(0, 7)}</span> on {day(newer.created_at)}.
        </>
      ),
      still: `This page stays on version ${summary.latest_version} until you choose.`,
      acts: [
        { label: 'Show it', on: () => location.reload(), primary: true },
        { label: 'Later', on: dismiss(`newer-${newer.id}`) },
      ],
    })
  }
  notices.push(...given.filter((n) => !dismissed.includes(n.id)).map((n) => ({ ...n, acts: n.acts ?? [{ label: 'Got it', on: dismiss(n.id) }] })))

  const top = notices[0]
  return top ? <Band key={top.id} n={top} /> : null
}

function Band({ n }: { n: Notice }) {
  const [open, setOpen] = useState(!n.fold)
  return (
    <div className="page-notice" role="status" aria-live="polite">
      <span className={`page-notice-glyph glyph-${n.glyph}`} aria-hidden="true">
        {glyphs[n.glyph]}
      </span>
      <div className="page-notice-body">
        <div className="page-notice-head">
          <span className="page-notice-what">{n.what}</span>
          {n.fold && (
            <button className="link-btn" aria-expanded={open} onClick={() => setOpen((o) => !o)}>
              {open ? 'Less' : 'What still works'}
            </button>
          )}
        </div>
        {open && n.still && <div>{n.still}</div>}
        {open && n.todo && <div className="muted">{n.todo}</div>}
        {open && n.acts && n.acts.length > 0 && (
          <div className="page-notice-acts">
            {n.acts.map((a) => (
              <button key={a.label} className={a.primary ? 'btn btn-ink' : 'link-btn'} onClick={a.on}>
                {a.label}
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

/**
 * A version newer than the one this page loaded — locally, after `archdoc generate` in a terminal
 * (§4.8, "a newer version arrives"). Asked every 15 seconds while the page is visible, in a place
 * with history; nothing on the page changes until the reader says so.
 */
function useNewer(summary: Summary, updatingSince?: string): Version | undefined {
  const source = useSource()
  const can = useCapabilities()
  const [newer, setNewer] = useState<Version>()
  useEffect(() => {
    if (!can.history) return
    let live = true
    const look = async () => {
      if (document.hidden) return
      try {
        const s: Summary = await (await source.get('/api/summary')).json()
        if (s.latest_version <= summary.latest_version) return
        const versions: Version[] = await (await source.get('/api/versions')).json()
        const v = versions.find((x) => x.id === s.latest_version)
        if (live && v) setNewer(v)
      } catch {
        /* the connection's notice says so */
      }
    }
    const t = setInterval(look, 15000)
    // An update that has just ended: look now, rather than in up to fifteen seconds.
    if (updatingSince === undefined) void look()
    return () => {
      live = false
      clearInterval(t)
    }
  }, [can.history, source, summary.latest_version, updatingSince])
  return newer
}

interface Busy {
  by: string
  since: string
  doing?: string
  found?: string[]
}

/** A run updating the repository now — from the terminal, on this machine — asked every few seconds. */
function useUpdating(): Busy | undefined {
  const source = useSource()
  const can = useCapabilities()
  const [busy, setBusy] = useState<Busy>()
  useEffect(() => {
    if (can.place !== 'local') return
    let live = true
    const look = async () => {
      if (document.hidden) return
      try {
        const b = await (await source.get('/api/updating')).json()
        if (live) setBusy(b.running ? b : undefined)
      } catch {
        /* the connection's notice says so */
      }
    }
    void look()
    const t = setInterval(look, 4000)
    return () => {
      live = false
      clearInterval(t)
    }
  }, [can.place, source])
  return busy
}
