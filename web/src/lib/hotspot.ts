export interface PercentBox {
  left: number
  top: number
  width: number
  height: number
}

/**
 * Converts a [x, y, w, h] box in image pixels to percentages of the image,
 * using the real image size reported by the API, so overlays are placed
 * before the image loads and scale with it. Parts outside the image are
 * clipped. Returns null for a malformed box or unknown image size.
 */
export function boxToPercent(box: unknown, imgW: number, imgH: number): PercentBox | null {
  if (!Array.isArray(box) || box.length !== 4 || !(imgW > 0) || !(imgH > 0)) return null
  const [x, y, w, h] = box as unknown[]
  if (![x, y, w, h].every((v) => typeof v === 'number' && Number.isFinite(v))) return null
  const x0 = Math.max(0, x as number)
  const y0 = Math.max(0, y as number)
  const x1 = Math.min(imgW, (x as number) + (w as number))
  const y1 = Math.min(imgH, (y as number) + (h as number))
  if (x1 <= x0 || y1 <= y0) return null
  return {
    left: (x0 / imgW) * 100,
    top: (y0 / imgH) * 100,
    width: ((x1 - x0) / imgW) * 100,
    height: ((y1 - y0) / imgH) * 100,
  }
}

/** Reads the hotspot number from a location hash such as "h3" or "#h3". */
export function hotspotFromHash(hash: string | undefined): number | null {
  const m = /^#?h(\d+)$/.exec(hash ?? '')
  return m ? Number(m[1]) : null
}
