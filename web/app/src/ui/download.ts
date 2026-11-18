// Saving what Download offers: a file the place serves, or a PNG drawn here from the SVG — the same
// figure, at twice its size so it stays sharp.

/** Saves a blob under a file name, as a browser download. */
export function save(blob: Blob, name: string): void {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = name
  document.body.appendChild(a)
  a.click()
  a.remove()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}

/** An SVG drawn onto a canvas, as a PNG, at `scale` times its own size. */
export async function pngOf(svg: string, scale = 2): Promise<Blob> {
  const size = /<svg[^>]*\bwidth="([\d.]+)"[^>]*\bheight="([\d.]+)"/.exec(svg)
  const w = size ? Number(size[1]) : 1200
  const h = size ? Number(size[2]) : 800
  const url = URL.createObjectURL(new Blob([svg], { type: 'image/svg+xml' }))
  try {
    const img = new Image()
    await new Promise<void>((ok, no) => {
      img.onload = () => ok()
      img.onerror = () => no(new Error('the SVG could not be drawn'))
      img.src = url
    })
    const canvas = document.createElement('canvas')
    canvas.width = Math.round(w * scale)
    canvas.height = Math.round(h * scale)
    const ctx = canvas.getContext('2d')
    if (!ctx) throw new Error('no canvas to draw on')
    ctx.drawImage(img, 0, 0, canvas.width, canvas.height)
    return await new Promise<Blob>((ok, no) => canvas.toBlob((b) => (b ? ok(b) : no(new Error('no PNG'))), 'image/png'))
  } finally {
    URL.revokeObjectURL(url)
  }
}
