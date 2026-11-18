// Home on a return visit (platform-surface §17.3; app-design-screens §3): what is new since this
// reader last looked, and the plan's verdict in one line. Pure: the screen draws what this says, and
// the tests read it.

import type { PlanCheck } from '../../api'

/**
 * What the versions since the last look did, in one sentence:
 * "2 new versions since you last looked changed the architecture: added 3 ways in; changed 1 detail."
 */
export function sinceSentence(newer: number, structural: boolean, clauses: string[]): string {
  const versions = newer === 1 ? '1 new version' : `${newer} new versions`
  if (!structural) return `${versions} since you last looked; ${newer === 1 ? 'it didn’t change' : 'none changed'} the architecture.`
  return `${versions} since you last looked changed the architecture: ${clauses.join('; ')}.`
}

/** The plan check, in one line: "The plan asked for 3 things. 2 look built; 1 wasn’t found." Never a score. */
export function planLine(check: PlanCheck): string {
  const n = check.verdicts.length
  const count = (s: string) => check.verdicts.filter((v) => v.status === s).length
  const built = count('built'), missing = count('missing'), unclear = count('unclear')
  const said = [
    built ? `${built} ${built === 1 ? 'looks' : 'look'} built` : '',
    missing ? `${missing} ${missing === 1 ? 'wasn’t' : 'weren’t'} found` : '',
    unclear ? `${unclear} ${unclear === 1 ? 'is' : 'are'} unclear` : '',
  ].filter(Boolean)
  return `The plan asked for ${n === 1 ? 'one thing' : `${n} things`}.${said.length ? ` ${said.join('; ')}.` : ''}`
}
