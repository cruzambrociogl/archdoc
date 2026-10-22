import type { ReactNode } from 'react'
import { useTheme } from '../theme'
import { themeGlyph, themeLabel } from './TopBar'

/** Where archdoc runs on a machine of your own: the engine, which is public. */
export const engineHome = 'https://github.com/cruzambrociogl/archdoc'

/**
 * The top bar before there is a repository to show — Start, and a read on its way (App.dc.html,
 * "Landing" and "Generating"): the wordmark, what is being read, and the theme. No search, no share
 * and no menu, because none of the pages exist yet.
 */
export function WebsiteBar(props: { onStart: () => void; children?: ReactNode }) {
  const [theme, nextTheme] = useTheme()
  return (
    <header className="topbar">
      <button className="brand-mark" onClick={props.onStart}>
        archdoc
      </button>
      {props.children}
      <div className="topbar-spacer" />
      <button className="icon-btn" onClick={nextTheme} title={themeLabel[theme]} aria-label={themeLabel[theme]}>
        {themeGlyph[theme]}
      </button>
    </header>
  )
}
