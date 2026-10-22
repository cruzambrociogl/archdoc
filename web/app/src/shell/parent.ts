import { createContext, useContext } from 'react'

/** One level above the page: its name in the breadcrumb, and how to go there. */
export interface Crumb {
  label: string
  open: () => void
}

/**
 * The levels above a repository, where there are any — Home, its system (platform-surface §4.3,
 * "the breadcrumb carries every level"). The top bar puts them first. `repo` is the repository's
 * name there, which the breadcrumb says rather than the model's own: two repositories of one system
 * may both call themselves after it.
 */
export const ParentContext = createContext<{ crumbs: Crumb[]; repo: string } | null>(null)

export const useParent = () => useContext(ParentContext)
