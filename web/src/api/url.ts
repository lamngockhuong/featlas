// Every URL the app requests goes through this module, so the static export
// (a folder of JSON files next to index.html) and the live server share the
// same code. The export marks itself with <meta name="muninn-static">.

export function isStatic(): boolean {
  return (
    typeof document !== 'undefined' && document.querySelector('meta[name="muninn-static"]') !== null
  )
}

function relative(path: string): string {
  return new URL(path, document.baseURI).href
}

/** URL of an API resource; `name` is a path below /api, e.g. "entities/screen/claim-list". */
export function apiUrl(name: string): string {
  const clean = name.replace(/^\/+/, '')
  return isStatic() ? relative(`api/${clean}.json`) : `/api/${clean}`
}

/** URL of an entity detail resource. */
export function entityUrl(kind: string, id: string): string {
  return apiUrl(`entities/${encodeURIComponent(kind)}/${encodeURIComponent(id)}`)
}

/** Maps /content/... URLs from the API to a relative URL in static mode. */
export function contentUrl(url: string): string {
  if (isStatic() && url.startsWith('/content/')) return relative(url.slice(1))
  return url
}
