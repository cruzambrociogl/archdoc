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
export const ParentContext = createContext<{ crumbs: Crumb[]; repo: string; line?: Line } | null>(null)

/**
 * On the website, which line of the repository is shown — its default branch, another branch, a tag
 * or one commit (shell/line.ts) — and the others archdoc has read, each opening at its address.
 */
export interface Line {
  label: string
  kind: 'branch' | 'tag' | 'commit'
  others: (Crumb & { kind: 'branch' | 'tag' | 'commit' })[]
}

export const useParent = () => useContext(ParentContext)
