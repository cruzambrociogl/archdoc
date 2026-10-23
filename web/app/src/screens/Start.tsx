import { useCallback, useEffect, useRef, useState } from 'react'
import type { Runs } from '../source'
import { Refusal } from '../source'
import { WebsiteBar, engineHome } from '../shell/WebsiteBar'

/**
 * Start (platform-surface §5.1): the page the website's link box opens, `/new?repo=…`, which asks at
 * once — and a box of its own for anyone arriving without a link. It says what each refusal means;
 * an accepted link goes on to its run. At a repository's address no one has asked for (`suggest`),
 * it says so, with the link ready — and waits to be asked.
 */
export function Start(props: { repo?: string; suggest?: string; runs: Runs; onRun: (id: number, replace: boolean) => void; onStart: () => void }) {
  const { runs, onRun } = props
  const [link, setLink] = useState(props.repo ?? props.suggest ?? '')
  const [asking, setAsking] = useState(false)
  const [refused, setRefused] = useState<Error | null>(null)

  // Arriving with a link asks once, as if it had been pasted here; the run then takes this address's place.
  const ask = useCallback(
    async (l: string, replace: boolean) => {
      setAsking(true)
      setRefused(null)
      try {
        onRun(await runs.request(l), replace)
      } catch (e) {
        setRefused(e instanceof Error ? e : new Error(String(e)))
        setAsking(false)
      }
    },
    [runs, onRun],
  )
  const asked = useRef(false)
  useEffect(() => {
    if (props.repo && !asked.current) {
      asked.current = true
      void ask(props.repo, true)
    }
  }, [props.repo, ask])

  const kind = refused instanceof Refusal ? refused.kind : refused ? 'unavailable' : null
  return (
    <div className="shell">
      <WebsiteBar onStart={props.onStart} />
      <main className="shell-main page">
        <div className="start">
          <h1 className="start-claim">
            {props.suggest
              ? `archdoc hasn’t read ${props.suggest.replace('https://github.com/', '')} yet.`
              : 'The architecture of any repository, read from its code — every box and arrow cites the line that proves it.'}
          </h1>
          <form
            className="start-box"
            onSubmit={(e) => {
              e.preventDefault()
              if (link.trim()) void ask(link.trim(), false)
            }}
          >
            <input
              aria-label="GitHub repository link"
              placeholder="Paste a GitHub link — github.com/owner/repo"
              value={link}
              onChange={(e) => setLink(e.target.value)}
              autoFocus
              spellCheck={false}
            />
            <button className="btn btn-ink" type="submit" disabled={asking || !link.trim()}>
              {asking ? 'Asking GitHub…' : 'Show its architecture'}
            </button>
          </form>
          {kind && (
            <p className="start-msg" role="alert">
              {kind === 'not_github'
                ? 'That isn’t a link to a GitHub repository. It should look like github.com/immich-app/immich.'
                : kind === 'not_found'
                  ? 'GitHub shows no public repository there. It may be private, or not exist — private repositories need an account, which the website doesn’t have yet.'
                  : `archdoc couldn’t take the request: ${refused!.message}`}
            </p>
          )}
          <p className="start-note">
            No account needed for public repositories. On your own machine archdoc has no limits and sends nothing:{' '}
            <a href={engineHome} target="_blank" rel="noopener noreferrer">{engineHome.replace('https://', '')}</a>.
          </p>
        </div>
      </main>
    </div>
  )
}
