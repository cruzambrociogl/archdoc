import { useEffect, useRef, useState } from 'react'
import type { SignedIn } from '../source'
import { useSource } from '../source'

// Signing out reloads the page, or leaves it for one that needs no account; this says, on the page
// that follows, that it worked — the avatar turning into Sign in is easy to miss.
const signedOutKey = 'archdoc:signed-out'

function remember() {
  try {
    sessionStorage.setItem(signedOutKey, '1')
  } catch {
    // No storage: signing out still works; it just goes unsaid.
  }
}

function justSignedOut(): boolean {
  try {
    const was = sessionStorage.getItem(signedOutKey) === '1'
    sessionStorage.removeItem(signedOutKey)
    return was
  } catch {
    return false
  }
}

/**
 * The bar's last item where the place has accounts: Sign in, or the person — their initials, or their
 * picture — and their menu: what the place lists, then Sign out. Where the place has none, nothing.
 */
export function AccountButton() {
  const account = useSource().account
  const [me, setMe] = useState<SignedIn | null>()
  const [open, setOpen] = useState(false)
  const [told, setTold] = useState(false)
  const anchor = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!justSignedOut()) return
    setTold(true)
    const t = setTimeout(() => setTold(false), 3200)
    return () => clearTimeout(t)
  }, [])

  useEffect(() => {
    let live = true
    account?.me().then(
      (m) => live && setMe(m),
      () => live && setMe(null),
    )
    return () => {
      live = false
    }
  }, [account])

  useEffect(() => {
    if (!open) return
    const away = (e: MouseEvent) => {
      if (anchor.current && !anchor.current.contains(e.target as Node)) setOpen(false)
    }
    const esc = (e: KeyboardEvent) => e.key === 'Escape' && setOpen(false)
    document.addEventListener('mousedown', away)
    document.addEventListener('keydown', esc)
    return () => {
      document.removeEventListener('mousedown', away)
      document.removeEventListener('keydown', esc)
    }
  }, [open])

  const toast = told && (
    <div className="toast" role="status">
      You’ve signed out.
    </div>
  )
  if (!account || me === undefined) return toast || null
  if (!me) {
    return (
      <>
        <a className="btn btn-outline account-signin" href={account.signIn(location.pathname + location.search + location.hash)}>
          Sign in
        </a>
        {toast}
      </>
    )
  }
  const signOut = async () => {
    const next = await account.signOut()
    remember()
    if (next) location.assign(next)
    else location.reload()
  }
  return (
    <div className="pop-anchor" ref={anchor}>
      <button className={`avatar-btn ${open ? 'on' : ''}`} onClick={() => setOpen(!open)} aria-label={`Signed in as ${me.name}`} aria-expanded={open} title={me.name}>
        {me.avatar ? <img className="avatar-img" src={me.avatar} alt="" /> : <span className="avatar-initials">{me.initials}</span>}
      </button>
      {open && (
        <div className="pop more-pop account-pop">
          <div className="account-who">
            <span className="more-label">{me.name}</span>
            <span className="more-sub">{me.email}</span>
          </div>
          {me.links.map((l) => (
            <a key={l.href} className="more-item" href={l.href}>
              <span className="more-label">{l.label}</span>
              {l.note && <span className="more-right">{l.note}</span>}
            </a>
          ))}
          <button className="more-item account-out" onClick={() => void signOut()}>
            <span className="more-label">Sign out</span>
          </button>
        </div>
      )}
    </div>
  )
}
