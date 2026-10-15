import { useEffect, useState } from 'react'
import { useT } from '../lib/prefs'

async function copyText(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch {
    // Clipboard API needs a secure context; fall back for plain http on a LAN.
    const area = document.createElement('textarea')
    area.value = text
    area.style.position = 'fixed'
    area.style.opacity = '0'
    document.body.appendChild(area)
    area.select()
    const ok = document.execCommand('copy')
    area.remove()
    return ok
  }
}

export function CopyButton({ text, label }: { text: string; label?: string }) {
  const t = useT()
  const [copied, setCopied] = useState(false)
  useEffect(() => {
    if (!copied) return
    const timer = setTimeout(() => setCopied(false), 1500)
    return () => clearTimeout(timer)
  }, [copied])
  return (
    <button
      type="button"
      onClick={async () => setCopied(await copyText(text))}
      aria-label={label ?? t('flow.copy')}
      className="rounded border border-slate-300 px-1.5 py-0.5 text-xs text-slate-600 hover:bg-slate-100 dark:border-slate-600 dark:text-slate-300 dark:hover:bg-slate-800"
    >
      <span aria-live="polite">{copied ? t('flow.copied') : t('flow.copy')}</span>
    </button>
  )
}
