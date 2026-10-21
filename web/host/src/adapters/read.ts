/** Fetches an address; a failed answer becomes an Error carrying the server's own message. */
export async function read(url: string): Promise<Response> {
  const r = await fetch(url)
  if (!r.ok) {
    let msg = r.statusText
    try {
      msg = (await r.json()).error ?? msg
    } catch {
      /* not JSON: keep the status text */
    }
    throw new Error(msg)
  }
  return r
}
