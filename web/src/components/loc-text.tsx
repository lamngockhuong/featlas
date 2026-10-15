import { useContentLang, useT } from '../lib/prefs'

/** Marker shown next to content displayed in a fallback language. */
export function Untranslated({ lang }: { lang: string }) {
  const t = useT()
  const label = t('untranslated', { lang: lang.toUpperCase() })
  return (
    <span
      title={label}
      className="ml-1 inline-block rounded border border-dashed border-slate-400 px-1 align-middle text-[10px] font-normal leading-4 text-slate-500 uppercase dark:text-slate-400"
    >
      <span aria-hidden="true">{lang}</span>
      <span className="sr-only">{label}</span>
    </span>
  )
}

/** Renders a localized value in the content language, with a fallback marker. */
export function Loc({ v, fallback = '' }: { v: unknown; fallback?: string }) {
  const { loc } = useContentLang()
  const p = loc(v)
  if (!p.text) return <>{fallback}</>
  return (
    <>
      {p.text}
      {p.fallback && <Untranslated lang={p.fallback} />}
    </>
  )
}
