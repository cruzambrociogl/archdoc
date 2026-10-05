// The viewer's theme choice. A per-viewer convenience, so browser storage is enough — and every
// access is guarded, because storage can be missing or blocked (a private window, a file:// page).

import { useEffect, useState } from 'react'

export type Theme = 'system' | 'light' | 'dark'
const key = 'archdoc.theme'

function read(): Theme {
  try {
    const t = localStorage.getItem(key)
    return t === 'light' || t === 'dark' ? t : 'system'
  } catch {
    return 'system'
  }
}

function apply(t: Theme) {
  const root = document.documentElement
  if (t === 'system') delete root.dataset.theme
  else root.dataset.theme = t
}

export function useTheme(): [Theme, () => void] {
  const [theme, setTheme] = useState<Theme>(read)
  useEffect(() => {
    apply(theme)
    try {
      if (theme === 'system') localStorage.removeItem(key)
      else localStorage.setItem(key, theme)
    } catch {
      /* not persisted; the choice still holds for this page */
    }
  }, [theme])
  const next = () => setTheme((t) => (t === 'system' ? 'light' : t === 'light' ? 'dark' : 'system'))
  return [theme, next]
}
