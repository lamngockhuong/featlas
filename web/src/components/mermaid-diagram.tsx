import { useEffect, useId, useState } from 'react'
import { usePrefs } from '../lib/prefs'

// mermaid is large, so it is imported only when a page has a diagram.
let loader: Promise<typeof import('mermaid').default> | null = null
function loadMermaid() {
  loader ??= import('mermaid').then((m) => m.default)
  return loader
}

export function MermaidDiagram({ code }: { code: string }) {
  const { theme } = usePrefs()
  const id = `mmd-${useId().replace(/[^a-zA-Z0-9]/g, '')}`
  const [svg, setSvg] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    loadMermaid()
      .then(async (mermaid) => {
        mermaid.initialize({
          startOnLoad: false,
          securityLevel: 'strict',
          theme: theme === 'dark' ? 'dark' : 'default',
        })
        const out = await mermaid.render(`${id}-${theme}`, code)
        if (!cancelled) {
          setSvg(out.svg)
          setError(null)
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof Error ? err.message : String(err))
      })
    return () => {
      cancelled = true
    }
  }, [code, id, theme])

  if (error) {
    return (
      <pre className="overflow-x-auto rounded border border-red-300 p-3 text-xs" title={error}>
        {code}
      </pre>
    )
  }
  if (!svg) return <div className="h-24 animate-pulse rounded bg-slate-100 dark:bg-slate-800" />
  return (
    <div
      className="my-4 flex justify-center overflow-x-auto"
      // The SVG comes from mermaid with securityLevel "strict", which sanitizes labels.
      // biome-ignore lint/security/noDangerouslySetInnerHtml: rendered mermaid SVG
      dangerouslySetInnerHTML={{ __html: svg }}
    />
  )
}
