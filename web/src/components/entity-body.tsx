import { useContentLang } from '../lib/prefs'
import { Untranslated } from './loc-text'
import { Markdown } from './markdown'

/** The Markdown body of an entity in the content language, with a fallback marker. */
export function EntityBody({ bodies }: { bodies: Record<string, string> }) {
  const { loc } = useContentLang()
  const body = loc(bodies)
  if (!body.text.trim()) return null
  return (
    <div className="mt-6">
      {body.fallback && (
        <p className="mb-2 text-xs text-slate-500">
          <Untranslated lang={body.fallback} />
        </p>
      )}
      <Markdown text={body.text} />
    </div>
  )
}
