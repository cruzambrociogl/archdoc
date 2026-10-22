import { createContext, useContext } from 'react'

/**
 * The level above a repository, where there is one — its system (platform-surface §4.3, "the
 * breadcrumb carries every level"). The top bar puts it first, and it opens that level. `repo` is the
 * repository's name in it, which the breadcrumb says rather than the model's own name: two
 * repositories of one system may both call themselves after it.
 */
export const ParentContext = createContext<{ label: string; repo: string; open: () => void } | null>(null)

export const useParent = () => useContext(ParentContext)
