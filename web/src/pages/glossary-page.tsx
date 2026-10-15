import { Link, useLocation } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import { useGlossary } from '../api/client'
import type { Kind } from '../api/types'
import { EntityLink } from '../components/entity-link'
import { useEntityTitle } from '../components/entity-refs'
import { Loc } from '../components/loc-text'
import { ErrorBox, Loading } from '../components/query-state'
import { Muted } from '../components/section'
import { useContentLang, useT } from '../lib/prefs'

export function GlossaryPage() {
  const t = useT()
  const { text } = useContentLang()
  const title = useEntityTitle()
  const q = useGlossary()
  const hash = useLocation({ select: (l) => l.hash })
  const [filter, setFilter] = useState('')

  useEffect(() => {
    if (q.data && hash) document.getElementById(hash)?.scrollIntoView({ block: 'center' })
  }, [q.data, hash])

  if (q.isPending) return <Loading />
  if (q.isError) return <ErrorBox error={q.error} retry={() => q.refetch()} />

  const needle = filter.trim().toLowerCase()
  const terms = [...q.data]
    .sort((a, b) => text(a.term).localeCompare(text(b.term)))
    .filter((g) => {
      if (!needle) return true
      const hay = [g.id, ...Object.values(g.term), ...Object.values(g.meaning)]
      return hay.some((s) => s.toLowerCase().includes(needle))
    })
  const termName = (id: string) => {
    const g = q.data.find((x) => x.id === id)
    return g ? text(g.term) || id : id
  }

  return (
    <div className="mx-auto max-w-5xl px-4 py-6 sm:px-6">
      <h1 className="text-2xl font-semibold">{t('nav.glossary')}</h1>
      <input
        type="search"
        value={filter}
        onChange={(e) => setFilter(e.target.value)}
        placeholder={t('glossary.filter')}
        aria-label={t('glossary.filter')}
        className="input mt-4 w-full max-w-sm"
      />
      {terms.length === 0 ? (
        <div className="mt-6">
          <Muted>{t('glossary.empty')}</Muted>
        </div>
      ) : (
        <div className="mt-4 overflow-x-auto">
          <table className="table">
            <thead>
              <tr>
                <th>{t('glossary.term')}</th>
                <th>{t('glossary.meaning')}</th>
                <th>{t('glossary.related')}</th>
                <th>{t('glossary.usedIn')}</th>
              </tr>
            </thead>
            <tbody>
              {terms.map((g) => {
                const users = g.refsTo.filter((r) => r.kind !== 'term')
                return (
                  <tr
                    key={g.id}
                    id={`term-${g.id}`}
                    className={hash === `term-${g.id}` ? 'bg-sky-50 dark:bg-sky-950/40' : ''}
                  >
                    <td className="font-medium">
                      <Loc v={g.term} fallback={g.id} />
                    </td>
                    <td>
                      <Loc v={g.meaning} />
                    </td>
                    <td>
                      {(g.related ?? []).map((r, i) => (
                        <span key={r}>
                          {i > 0 && ', '}
                          <Link to="/glossary" hash={`term-${r}`} className="link">
                            {termName(r)}
                          </Link>
                        </span>
                      ))}
                    </td>
                    <td>
                      {users.map((r, i) => (
                        <span key={`${r.kind}/${r.id}`}>
                          {i > 0 && ', '}
                          <EntityLink kind={r.kind as Kind} id={r.id} className="link">
                            {title(r.kind as Kind, r.id)}
                          </EntityLink>
                        </span>
                      ))}
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
