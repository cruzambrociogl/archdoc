import { useEffect, useRef, useState } from 'react'
import type { SignedIn } from '../source'
import { useSource } from '../source'

/**
 * The bar's last item where the place has accounts: Sign in, or the person — their initials, or their
 * picture — and their menu: what the place lists, then Sign out. Where the place has none, nothing.
 */
export function AccountButton() {
  const account = useSource().account
  const [me, setMe] = useState<SignedIn | null>()
  const [open, setOpen] = useState(false)
  const anchor = useRef<HTMLDivElement>(null)

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

  if (!account || me === undefined) return null
  if (!me) {
    return (
      <a className="btn btn-outline account-signin" href={account.signIn(location.pathname + location.search + location.hash)}>
        Sign in
      </a>
    )
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
          <button className="more-item account-out" onClick={() => account.signOut().then(() => location.reload())}>
            <span className="more-label">Sign out</span>
          </button>
        </div>
      )}
    </div>
  )
}
