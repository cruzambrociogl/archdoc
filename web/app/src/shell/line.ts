// A repository's lines on the website (decisions, 23 Oct): what a read reads — its default branch,
// another branch, a tag, or one commit — each a history of its own, at an address made the way
// GitHub's links are: /{owner}/{name}, /{owner}/{name}/tree/{branch or tag}, /{owner}/{name}/commit/{hash}.
// A line is stored as '' (the default branch), 'heads/<branch>', 'tags/<tag>' or 'commit/<hash>'.

import type { At } from '../source'

export type LineKind = 'branch' | 'tag' | 'commit'

export const lineKind = (ref = ''): LineKind => (ref.startsWith('commit/') ? 'commit' : ref.startsWith('tags/') ? 'tag' : 'branch')

/** The line as people name it: the branch or tag, a commit's first seven characters; the default branch by its name when it is known. */
export function lineLabel(ref = '', defaultBranch?: string): string {
  if (ref === '') return defaultBranch || 'default branch'
  if (ref.startsWith('commit/')) return ref.slice('commit/'.length, 'commit/'.length + 7)
  return ref.slice(ref.indexOf('/') + 1)
}

/** What an address asks for, from a stored line. */
export function atOf(ref = ''): At | undefined {
  if (ref === '') return undefined
  if (ref.startsWith('commit/')) return { commit: ref.slice('commit/'.length) }
  return { tree: ref.slice(ref.indexOf('/') + 1) }
}

/** The website's address of a repository at a line, or at what an address asked for. */
export function address(owner: string, name: string, at?: At): string {
  const base = `/${owner}/${name}`
  if (at?.commit) return `${base}/commit/${at.commit}`
  if (at?.tree) return `${base}/tree/${at.tree.split('/').map(encodeURIComponent).join('/')}`
  return base
}
