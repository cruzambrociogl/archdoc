/**
 * Puts text on the clipboard. Where the clipboard API is refused — a page opened from a file, an
 * older browser — or does not answer within a second, a hidden text box is copied instead. Says
 * whether it worked.
 */
export async function copyText(text: string): Promise<boolean> {
  try {
    const late = new Promise<never>((_, no) => setTimeout(() => no(new Error('no answer')), 1000))
    await Promise.race([navigator.clipboard.writeText(text), late])
    return true
  } catch {
    const t = document.createElement('textarea')
    t.value = text
    t.setAttribute('readonly', '')
    t.style.position = 'fixed'
    t.style.opacity = '0'
    document.body.appendChild(t)
    t.select()
    let ok = false
    try {
      ok = document.execCommand('copy')
    } catch {
      ok = false
    }
    t.remove()
    return ok
  }
}
