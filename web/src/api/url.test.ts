import { afterEach, describe, expect, it } from 'vitest'
import { apiUrl, contentUrl, entityUrl, isStatic } from './url'

function setStatic(on: boolean) {
  document.head.querySelector('meta[name="muninn-static"]')?.remove()
  if (on) {
    const m = document.createElement('meta')
    m.name = 'muninn-static'
    document.head.appendChild(m)
  }
}

afterEach(() => setStatic(false))

describe('api urls', () => {
  it('uses the live API by default', () => {
    expect(isStatic()).toBe(false)
    expect(apiUrl('project')).toBe('/api/project')
    expect(entityUrl('screen', 'claim-list')).toBe('/api/entities/screen/claim-list')
    expect(contentUrl('/content/screens/a/shots/x.webp')).toBe('/content/screens/a/shots/x.webp')
  })

  it('reads JSON files relative to the document in a static export', () => {
    setStatic(true)
    const base = new URL('.', document.baseURI).href
    expect(apiUrl('project')).toBe(`${base}api/project.json`)
    expect(entityUrl('flow', 'submit-claim')).toBe(`${base}api/entities/flow/submit-claim.json`)
    expect(contentUrl('/content/screens/a/shots/x.webp')).toBe(
      `${base}content/screens/a/shots/x.webp`,
    )
    expect(contentUrl('https://example.com/x.png')).toBe('https://example.com/x.png')
  })
})
