// The look of the C4 diagrams (platform-surface §16.1, the Classic C4 switch): archdoc's, or C4's
// usual blues. It is in the address, so a link shows what its sender saw; and each viewer's last
// choice is remembered for when an address does not say. Storage may be blocked; then archdoc's.

const KEY = 'archdoc.look'

export type Look = 'archdoc' | 'classic'

export function rememberedLook(): Look {
  try {
    return localStorage.getItem(KEY) === 'classic' ? 'classic' : 'archdoc'
  } catch {
    return 'archdoc'
  }
}

export function rememberLook(look: Look): void {
  try {
    localStorage.setItem(KEY, look)
  } catch {
    // Blocked: the address still carries it.
  }
}
