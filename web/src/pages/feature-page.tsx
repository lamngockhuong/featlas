import { Link, useParams } from '@tanstack/react-router'
import { useEntity, useGlossary } from '../api/client'
import type { FeatureFront } from '../api/types'
import { useConfigLabels } from '../components/config-labels'
import { EntityBody } from '../components/entity-body'
import { EntityHeader } from '../components/entity-header'
import { EntityList, useEntityMap } from '../components/entity-refs'
import { Loc } from '../components/loc-text'
import { ErrorBox, Loading } from '../components/query-state'
import { Muted, Section } from '../components/section'
import { ShotFrame } from '../components/shot-frame'
import { SourceList } from '../components/source-list'
import { StatusBadge } from '../components/status-badge'
import { useContentLang, useT } from '../lib/prefs'

export function FeaturePage() {
  const t = useT()
  const { text } = useContentLang()
  const { id } = useParams({ from: '/features/$id' })
  const q = useEntity('feature', id)
  const entities = useEntityMap()
  const glossary = useGlossary()
  const { areas, roles } = useConfigLabels()
  if (q.isPending) return <Loading />
  if (q.isError) return <ErrorBox error={q.error} retry={() => q.refetch()} />
  const d = q.data
  const front = d.front as FeatureFront
  const terms = (front.terms ?? []).map((tid) => ({
    id: tid,
    term: glossary.data?.find((g) => g.id === tid),
  }))

  return (
    <article className="mx-auto max-w-6xl px-4 py-6 sm:px-6">
      <EntityHeader
        entity={d}
        issues={d.issues}
        meta={
          <>
            {d.area && (
              <span>
                <span className="text-slate-500">{t('review.area')}: </span>
                {text(areas.get(d.area)) || d.area}
              </span>
            )}
            {d.roles.length > 0 && (
              <span>
                <span className="text-slate-500">{t('entity.roles')}: </span>
                {d.roles.map((r) => text(roles.get(r)) || r).join(', ')}
              </span>
            )}
            {d.owners.length > 0 && (
              <span>
                <span className="text-slate-500">{t('entity.owners')}: </span>
                {d.owners.join(', ')}
              </span>
            )}
          </>
        }
      />

      <EntityBody bodies={d.bodies} />

      <Section title={t('entity.screens')}>
        {front.screens?.length ? (
          <ul className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {front.screens.map((sid) => {
              const s = entities.get(`screen/${sid}`)
              return (
                <li key={sid}>
                  <Link
                    to="/screens/$id"
                    params={{ id: sid }}
                    className="group block rounded-lg border border-slate-200 p-2 hover:border-sky-400 dark:border-slate-800"
                  >
                    <ShotFrame media={s?.thumb} alt="" />
                    <span className="mt-2 flex items-center justify-between gap-2 px-1">
                      <span className="text-sm font-medium group-hover:underline">
                        {s ? <Loc v={s.title} fallback={sid} /> : sid}
                      </span>
                      {s && <StatusBadge state={s.status.state} />}
                    </span>
                  </Link>
                </li>
              )
            })}
          </ul>
        ) : (
          <Muted>{t('entity.none')}</Muted>
        )}
      </Section>

      <div className="grid gap-x-8 sm:grid-cols-2">
        <Section title={t('entity.flows')}>
          <EntityList kind="flow" ids={front.flows} />
        </Section>
        <Section title={t('entity.modules')}>
          <EntityList kind="module" ids={front.modules} />
        </Section>
        <Section title={t('entity.terms')}>
          {terms.length ? (
            <dl className="space-y-2 text-sm">
              {terms.map(({ id: tid, term }) => (
                <div key={tid}>
                  <dt>
                    <Link to="/glossary" hash={`term-${tid}`} className="link font-medium">
                      {term ? <Loc v={term.term} fallback={tid} /> : tid}
                    </Link>
                  </dt>
                  {term && (
                    <dd className="text-slate-600 dark:text-slate-400">
                      <Loc v={term.meaning} />
                    </dd>
                  )}
                </div>
              ))}
            </dl>
          ) : (
            <Muted>{t('entity.none')}</Muted>
          )}
        </Section>
        <Section title={t('entity.specs')}>
          {front.specs?.length ? (
            <ul className="space-y-1 text-sm">
              {front.specs.map((s) => (
                <li key={s.url}>
                  <a href={s.url} target="_blank" rel="noreferrer" className="link">
                    {s.label}
                  </a>
                </li>
              ))}
            </ul>
          ) : (
            <Muted>{t('entity.none')}</Muted>
          )}
        </Section>
      </div>

      <Section title={t('entity.code')}>
        <SourceList items={d.sources} />
      </Section>
    </article>
  )
}
