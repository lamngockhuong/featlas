import { describe, expect, it } from 'vitest'
import { boxToPercent, hotspotFromHash } from './hotspot'

describe('boxToPercent', () => {
  it('converts a pixel box to percentages of the real image size', () => {
    const p = boxToPercent([969, 124, 107, 38], 1366, 768)
    expect(p).not.toBeNull()
    expect(p?.left).toBeCloseTo((969 / 1366) * 100, 6)
    expect(p?.top).toBeCloseTo((124 / 768) * 100, 6)
    expect(p?.width).toBeCloseTo((107 / 1366) * 100, 6)
    expect(p?.height).toBeCloseTo((38 / 768) * 100, 6)
  })

  it('maps the whole image to 0..100', () => {
    expect(boxToPercent([0, 0, 800, 600], 800, 600)).toEqual({
      left: 0,
      top: 0,
      width: 100,
      height: 100,
    })
  })

  it('clips boxes that extend past the image edge', () => {
    expect(boxToPercent([700, 500, 200, 200], 800, 600)).toEqual({
      left: 87.5,
      top: (500 / 600) * 100,
      width: 12.5,
      height: (100 / 600) * 100,
    })
  })

  it('rejects malformed boxes and unknown image sizes', () => {
    expect(boxToPercent([1, 2, 3], 100, 100)).toBeNull()
    expect(boxToPercent('1,2,3,4', 100, 100)).toBeNull()
    expect(boxToPercent([1, 2, 3, Number.NaN], 100, 100)).toBeNull()
    expect(boxToPercent([1, 2, 3, 4], 0, 100)).toBeNull()
    expect(boxToPercent([200, 200, 10, 10], 100, 100)).toBeNull()
  })
})

describe('hotspotFromHash', () => {
  it('reads #h<n> anchors', () => {
    expect(hotspotFromHash('h3')).toBe(3)
    expect(hotspotFromHash('#h12')).toBe(12)
    expect(hotspotFromHash('term-claim')).toBeNull()
    expect(hotspotFromHash(undefined)).toBeNull()
  })
})
