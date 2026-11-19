import { useCallback, useEffect, useRef, useState } from 'react'
import type { Branches, Runs, StartOffer } from '../source'
import { Refusal, useSource } from '../source'
import { WebsiteBar, engineHome } from '../shell/WebsiteBar'

/**
 * Start (platform-surface §5.1): the page the website's link box opens, `/new?repo=…`, which asks at
 * once — and a box of its own for anyone arriving without a link. It says what each refusal means;
 * an accepted link goes on to its run. At a repository's address no one has asked for (`suggest`),
 * it says so, with the link ready — and waits to be asked. A pasted link to a repository is looked up
 * at once, and when it has several branches the box offers them, the default chosen: one click still
 * reads the default (decisions, 23 Oct).
 */
export function Start(props: { repo?: string; suggest?: string; runs: Runs; onRun: (id: number, replace: boolean) => void; onStart: () => void }) {
  const { runs, onRun } = props
  const [link, setLink] = useState(props.repo ?? props.suggest ?? '')
  const [asking, setAsking] = useState(false)
  const [refused, setRefused] = useState<Error | null>(null)
  const choice = useBranches(runs, link)
  const [branch, setBranch] = useState('')
  useEffect(() => setBranch(choice?.default ?? ''), [choice])

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
              if (!link.trim()) return
              // Another branch than the default is read by its own link, as GitHub makes it.
              void ask(choice && branch && branch !== choice.default ? `https://github.com/${choice.owner}/${choice.name}/tree/${branch}` : link.trim(), false)
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
            {choice && choice.branches.length > 1 && (
              <label className="start-branch">
                <span>Branch</span>
                <select value={branch} onChange={(e) => setBranch(e.target.value)}>
                  {choice.branches.map((b) => (
                    <option key={b} value={b}>
                      {b === choice.default ? `${b} (default)` : b}
                    </option>
                  ))}
                  {choice.more && <option disabled>More: paste the branch’s own link</option>}
                </select>
              </label>
            )}
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
                  : kind === 'no_ref'
                    ? `${refused!.message}. A link to a branch, tag or commit is the one GitHub shows for it.`
                  : `archdoc couldn’t take the request: ${refused!.message}`}
            </p>
          )}
          <Offer />
          <p className="start-note">
            No account needed for public repositories. A branch, tag or commit: paste its GitHub link. On your own machine archdoc has no limits and sends nothing:{' '}
            <a href={engineHome} target="_blank" rel="noopener noreferrer">{engineHome.replace('https://', '')}</a>.
          </p>
        </div>
      </main>
    </div>
  )
}

/** What the place offers someone signed in, under the box: their spaces, and making a project. */
function Offer() {
  const account = useSource().account
  const [offer, setOffer] = useState<StartOffer | null>(null)
  const [choosing, setChoosing] = useState(false)
  const anchor = useRef<HTMLDivElement>(null)
  useEffect(() => {
    let live = true
    account?.me().then(
      (m) => live && setOffer(m?.start ?? null),
      () => live && setOffer(null),
    )
    return () => {
      live = false
    }
  }, [account])
  useEffect(() => {
    if (!choosing) return
    const away = (e: MouseEvent) => {
      if (anchor.current && !anchor.current.contains(e.target as Node)) setChoosing(false)
    }
    const esc = (e: KeyboardEvent) => e.key === 'Escape' && setChoosing(false)
    document.addEventListener('mousedown', away)
    document.addEventListener('keydown', esc)
    return () => {
      document.removeEventListener('mousedown', away)
      document.removeEventListener('keydown', esc)
    }
  }, [choosing])
  if (!offer) return null
  const choices = offer.create?.choices ?? []
  return (
    <section className="start-offer" aria-label={offer.heading}>
      <div className="start-offer-head">
        <h2 className="start-offer-title">{offer.heading}</h2>
        {choices.length === 1 && (
          <a className="btn btn-outline" href={choices[0].href}>
            {offer.create!.label}
          </a>
        )}
        {choices.length > 1 && (
          <div className="pop-anchor" ref={anchor}>
            <button className="btn btn-outline" aria-expanded={choosing} onClick={() => setChoosing(!choosing)}>
              {offer.create!.label}
            </button>
            {choosing && (
              <div className="pop more-pop start-offer-pop">
                <div className="account-who">
                  <span className="more-sub">Create it in</span>
                </div>
                {choices.map((c) => (
                  <a key={c.href} className="more-item" href={c.href}>
                    <span className="more-label">{c.label}</span>
                  </a>
                ))}
              </div>
            )}
          </div>
        )}
      </div>
      <ul className="start-offer-list">
        {offer.items.map((i) => (
          <li key={i.href}>
            <a href={i.href}>{i.label}</a>
            {i.note && <span className="start-offer-note">{i.note}</span>}
          </li>
        ))}
      </ul>
    </section>
  )
}

// A link to a repository itself — not to a branch, a tag or a commit, which name what to read.
const bareRepository = /^(https?:\/\/)?(www\.)?github\.com\/[A-Za-z0-9-]+\/[A-Za-z0-9._-]+?(\.git)?\/?$/i

/** The branches of the repository a link names, once typing stops; none for anything else, or when GitHub does not say. */
function useBranches(runs: Runs, link: string): Branches | null {
  const [found, setFound] = useState<{ link: string; branches: Branches } | null>(null)
  const l = link.trim()
  useEffect(() => {
    if (!bareRepository.test(l)) return
    let stale = false
    const t = setTimeout(() => {
      runs.branches(l).then(
        (b) => !stale && setFound({ link: l, branches: b }),
        () => !stale && setFound(null),
      )
    }, 400)
    return () => {
      stale = true
      clearTimeout(t)
    }
  }, [runs, l])
  return found?.link === l ? found.branches : null
}
