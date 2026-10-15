import { useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import MiniSearch from 'minisearch'
import { type KeyboardEvent, useEffect, useMemo, useRef, useState } from 'react'
import { searchIndexQuery } from '../api/client'
import type { Kind, SearchDoc } from '../api/types'
import { useContentLang, useT } from '../lib/prefs'
import { entityRoute } from './entity-link'

// MiniSearch reserves `id` in results for the document id, so the entity id
// is indexed and stored as `eid`.
type Indexed = Omit<SearchDoc, 'id'> & { eid: string; titles: string }

export interface Hit {
  key: string
  kind: SearchDoc['kind']
  id: string
  title: SearchDoc['title']
  route?: string
}

export function buildIndex(docs: SearchDoc[]): MiniSearch<Indexed> {
  const ms = new MiniSearch<Indexed>({
    idField: 'key',
    fields: ['titles', 'eid', 'route', 'text'],
    storeFields: ['kind', 'eid', 'title', 'route'],
    searchOptions: { boost: { titles: 4, eid: 2, route: 2 }, prefix: true, fuzzy: 0.2 },
  })
  ms.addAll(
    docs.map(({ id, ...d }) => ({ ...d, eid: id, titles: Object.values(d.title).join(' ') })),
  )
  return ms
}

export function search(index: MiniSearch<Indexed>, query: string, limit = 20): Hit[] {
  return index
    .search(query)
    .slice(0, limit)
    .map((r) => ({
      key: String(r.id),
      kind: r.kind as Hit['kind'],
      id: r.eid as string,
      title: r.title as Hit['title'],
      route: r.route as string | undefined,
    }))
}

export function SearchDialog({ onClose }: { onClose: () => void }) {
  const t = useT()
  const { text } = useContentLang()
  const navigate = useNavigate()
  const q = useQuery(searchIndexQuery)
  const [query, setQuery] = useState('')
  const [cursor, setCursor] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)
  const index = useMemo(() => (q.data ? buildIndex(q.data) : null), [q.data])
  const results = useMemo(
    () => (index && query.trim() ? search(index, query.trim()) : []),
    [index, query],
  )

  useEffect(() => {
    const prev = document.activeElement as HTMLElement | null
    inputRef.current?.focus()
    return () => prev?.focus()
  }, [])

  const open = (r: Hit) => {
    onClose()
    if (r.kind === 'term') void navigate({ to: '/glossary', hash: `term-${r.id}` })
    else void navigate({ to: entityRoute(r.kind as Kind), params: { id: r.id } })
  }

  const onKey = (e: KeyboardEvent) => {
    if (e.key === 'Escape') onClose()
    else if (e.key === 'ArrowDown') setCursor((c) => Math.min(results.length - 1, c + 1))
    else if (e.key === 'ArrowUp') setCursor((c) => Math.max(0, c - 1))
    else if (e.key === 'Enter' && results[cursor]) open(results[cursor])
    else return
    e.preventDefault()
  }

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center p-4 pt-[10vh]">
      <button
        type="button"
        tabIndex={-1}
        aria-label={t('search.close')}
        onClick={onClose}
        className="absolute inset-0 bg-slate-950/50"
      />
      <div
        role="dialog"
        aria-modal="true"
        aria-label={t('header.search')}
        className="relative w-full max-w-xl overflow-hidden rounded-lg bg-white shadow-2xl dark:bg-slate-900"
      >
        <input
          ref={inputRef}
          type="search"
          aria-label={t('header.search')}
          aria-controls="search-results"
          value={query}
          onChange={(e) => {
            setQuery(e.target.value)
            setCursor(0)
          }}
          onKeyDown={onKey}
          placeholder={t('search.placeholder')}
          className="w-full border-b border-slate-200 bg-transparent px-4 py-3 outline-none dark:border-slate-700"
        />
        <div className="max-h-[60vh] overflow-y-auto py-1" aria-live="polite">
          {q.isPending && <p className="px-4 py-2 text-sm text-slate-500">{t('app.loading')}</p>}
          {q.isError && <p className="px-4 py-2 text-sm text-red-600">{t('app.error')}</p>}
          {index && query.trim() && results.length === 0 && (
            <p className="px-4 py-2 text-sm text-slate-500">{t('search.empty')}</p>
          )}
          <ul id="search-results">
            {results.map((r, i) => (
              <li key={r.key}>
                <button
                  type="button"
                  aria-current={i === cursor ? 'true' : undefined}
                  onMouseEnter={() => setCursor(i)}
                  onClick={() => open(r)}
                  className={`flex w-full items-center gap-3 px-4 py-2 text-left text-sm ${
                    i === cursor ? 'bg-sky-50 dark:bg-sky-950' : ''
                  }`}
                >
                  <span className="w-20 shrink-0 text-xs text-slate-500">
                    {t(`kind.${r.kind}`)}
                  </span>
                  <span className="truncate font-medium">{text(r.title) || r.id}</span>
                  {r.route && (
                    <code className="ml-auto truncate text-xs text-slate-500">{r.route}</code>
                  )}
                </button>
              </li>
            ))}
          </ul>
        </div>
      </div>
    </div>
  )
}
