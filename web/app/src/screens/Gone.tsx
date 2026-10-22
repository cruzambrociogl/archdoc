import type { ReactNode } from 'react'
import { useEffect, useState } from 'react'
import type { ModelResponse, Version } from '../api'
import { useApi } from '../api'
import type { Route } from '../route'
import { useCapabilities, useSource } from '../source'

const day = (iso: string) => new Date(iso).toLocaleDateString(undefined, { day: 'numeric', month: 'short' })

type Go = (r: Partial<Route>, o?: { keep?: boolean }) => void

/**
 * A link to something that is not here (platform-surface §4.8, "a link to a version, part or page
 * that no longer exists"): what is missing, what replaced it if anything is known, and where to go.
 * Never a bare "not found".
 */
export function Gone(
  props:
    | { kind: 'page'; address: string; go: Go }
    | { kind: 'version'; v: number; versions: Version[]; name: string; go: Go }
    | { kind: 'part'; id: string; shown: number; name: string; go: Go; compact?: boolean },
) {
  const can = useCapabilities()
  const { go } = props
  let q = 'Where did it go?'
  let s = ''
  let s2: ReactNode = null
  let rows: { label: string; sub: string; on: () => void }[] = []

  if (props.kind === 'page') {
    s = 'There’s no page at this address.'
    s2 = (
      <>
        <span className="mono">#{props.address}</span> may come from an older archdoc, or from a mistyped link. Every page is in
        the menu; these answer most questions:
      </>
    )
    rows = [
      { label: 'Overview', sub: 'what this is', on: () => go({ screen: 'overview' }) },
      { label: 'Map', sub: 'how it is put together', on: () => go({ screen: 'explorer' }) },
      { label: 'Reference', sub: 'the complete detail', on: () => go({ screen: 'reference' }) },
    ]
  } else if (props.kind === 'version') {
    q = 'Which version is this?'
    const ids = props.versions.map((v) => v.id).sort((a, b) => a - b)
    s = `There’s no version ${props.v} of ${props.name} here.`
    s2 = !can.history
      ? can.place === 'hosted'
        ? 'Here archdoc shows the one version this read documented.'
        : 'This site keeps only the version it was built from, and what changed since the one before it.'
      : ids.length
        ? `archdoc has recorded ${ids.length === 1 ? `one version, version ${ids[0]}` : `versions ${ids[0]} to ${ids[ids.length - 1]}`}.`
        : 'archdoc has recorded no versions yet.'
    rows = [
      { label: 'The latest version', sub: 'what the code is now', on: () => go({ v: undefined }, { keep: true }) },
      ...(can.history ? [{ label: 'Every version', sub: 'in What changed', on: () => go({ screen: 'changes', v: undefined }) }] : []),
    ]
  } else {
    return <PartGone {...props} />
  }

  return <Page label={q} s={s} s2={s2} rows={rows} />
}

/** A part a link names that this version does not have: where it was last seen, when history can say. */
function PartGone(props: { id: string; shown: number; name: string; go: Go; compact?: boolean }) {
  const can = useCapabilities()
  const versions = useApi<Version[]>(can.history ? '/api/versions' : null).data
  const last = useLastSeen(props.id, props.shown, can.history ? versions : none)
  const label = props.id.replace(/^[a-z]+:/, '')
  const after = last.version && [...(versions ?? [])].sort((a, b) => a.id - b.id).find((v) => v.id > last.version!.id)

  let s2: ReactNode
  let rows: { label: string; sub: string; on: () => void }[] = []
  if (last.looking) s2 = 'Looking for it in the earlier versions…'
  else if (last.version) {
    const v = last.version
    s2 = (
      <>
        It was last in version {v.id},{' '}
        {v.commit ? (
          <>
            from commit <span className="mono">{v.commit.slice(0, 7)}</span>{' '}
          </>
        ) : (
          'recorded '
        )}
        on {day(v.created_at)}.
      </>
    )
    rows = [
      {
        label: `${last.name ?? label}, as it was`,
        sub: `in version ${v.id}`,
        // A component has a page of its own; any other part is shown where it stood, on the Map —
        // which, asked from the Map, is where the link already points.
        on: () =>
          last.kind === 'component' || props.compact ? props.go({ v: v.id }, { keep: true }) : props.go({ screen: 'explorer', level: 'container', focus: props.id, v: v.id }),
      },
      ...(after ? [{ label: `What changed in version ${after.id}`, sub: 'the change that removed it', on: () => props.go({ screen: 'changes', v: after.id }) }] : []),
    ]
  } else {
    s2 = can.history ? 'archdoc has no record of a part with that name in any version.' : 'This site keeps one version, so it can’t say where it went.'
    rows = [{ label: 'Reference', sub: 'every part, as this version has it', on: () => props.go({ screen: 'reference' }) }]
  }
  const s = `${label} isn’t in this version of ${props.name}.`
  if (props.compact) {
    // On the Map: a note over the picture, which stays, rather than a page in its place.
    return (
      <div className="gone-note" role="note">
        <div className="page-q">Where did it go?</div>
        <p>
          {s} {s2}
        </p>
        <div className="gone-note-acts">
          {rows.map((r) => (
            <button key={r.label} className="link-btn" onClick={r.on}>
              {r.label} →
            </button>
          ))}
        </div>
      </div>
    )
  }
  return <Page label="Where did it go?" s={s} s2={s2} rows={rows} />
}

function Page(props: { label: string; s: string; s2: ReactNode; rows: { label: string; sub: string; on: () => void }[] }) {
  return (
    <div className="reading gone">
      <div className="page-q">{props.label}</div>
      <p className="overview-answer">{props.s}</p>
      {props.s2 && <p className="gone-more">{props.s2}</p>}
      {props.rows.length > 0 && (
        <div className="doors">
          {props.rows.map((r) => (
            <button key={r.label} className="door" onClick={r.on}>
              <span className="door-title">{r.label}</span>
              <span className="door-count">→</span>
              <span className="door-what">{r.sub}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

const none: Version[] = []

/** The newest version before `shown` whose model has the element, read one version at a time. */
function useLastSeen(id: string, shown: number, versions: Version[] | undefined) {
  const source = useSource()
  const [found, setFound] = useState<{ looking: boolean; version?: Version; name?: string; kind?: string }>({ looking: versions !== none })
  useEffect(() => {
    if (!versions) return
    let live = true
    const older = versions.filter((v) => v.id < shown).sort((a, b) => b.id - a.id)
    ;(async () => {
      for (const v of older) {
        try {
          const m: ModelResponse = await (await source.get(`/api/model?version=${v.id}`)).json()
          const n = (m.model.nodes ?? []).find((x) => x.id === id)
          if (n) {
            if (live) setFound({ looking: false, version: v, name: n.name, kind: n.kind })
            return
          }
        } catch {
          break
        }
      }
      if (live) setFound({ looking: false })
    })()
    return () => {
      live = false
    }
  }, [id, shown, versions, source])
  return found
}
