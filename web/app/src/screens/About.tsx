import type { Session, Summary } from '../api'
import { when } from '../api'
import { useCapabilities } from '../source'
import type { Route } from '../route'
import { Coverage } from './Coverage'

/**
 * About these docs (platform-surface §4.4): the honesty of the product in one place — which commit
 * and version this is, what archdoc could and could not read, and, on this machine, what left it.
 * Reached from the Overview's trust sentence and from ⋯; not a menu entry.
 */
export function About(props: { summary: Summary; session?: Session; go: (r: Partial<Route>) => void }) {
  const { summary: s, session, go } = props
  const can = useCapabilities()
  const commit = session?.commit ?? s.commit
  const version = session?.version ?? s.latest_version

  return (
    <>
      <section className="about-head">
        <div className="page-q">Can I trust these docs?</div>
        <p className="about-facts">
          Version {version}
          {commit && (
            <>
              {' '}
              from commit <code>{commit.slice(0, 7)}</code>
            </>
          )}
          {session?.generated_at && <>, generated {when(session.generated_at)}</>}
          {s.source && (
            <>
              , read from <code>{s.source}</code>
            </>
          )}
          .
        </p>
        <p className="about-links">
          {can.runDetails && (
            <button className="link-btn" onClick={() => go({ screen: 'runs' })}>
              What left this machine{s.runs ? ` · ${s.runs} ${s.runs === 1 ? 'run' : 'runs'}` : ' · nothing'}
            </button>
          )}
          {/* On a published site there is no Settings: the corrections in force are listed here (§4.4). */}
          {can.place === 'published' && (
            <button className="link-btn" onClick={() => go({ screen: 'rules' })}>
              Corrections in force
            </button>
          )}
        </p>
      </section>
      <Coverage go={go} />
    </>
  )
}
