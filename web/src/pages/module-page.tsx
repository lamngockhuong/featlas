import { useParams } from '@tanstack/react-router'
import { useEntity } from '../api/client'
import type { ModuleFront } from '../api/types'
import { RoleNames } from '../components/config-labels'
import { EntityBody } from '../components/entity-body'
import { EntityHeader } from '../components/entity-header'
import { EntityList, refIds } from '../components/entity-refs'
import { Loc } from '../components/loc-text'
import { ErrorBox, Loading } from '../components/query-state'
import { Chip, Muted, Section } from '../components/section'
import { SourceList } from '../components/source-list'
import { useT } from '../lib/prefs'

export function ModulePage() {
  const t = useT()
  const { id } = useParams({ from: '/modules/$id' })
  const q = useEntity('module', id)
  if (q.isPending) return <Loading />
  if (q.isError) return <ErrorBox error={q.error} retry={() => q.refetch()} />
  const d = q.data
  const front = d.front as ModuleFront

  return (
    <article className="mx-auto max-w-5xl px-4 py-6 sm:px-6">
      <EntityHeader
        entity={d}
        issues={d.issues}
        meta={
          front.kind && (
            <span>
              <span className="text-slate-500">{t('module.kind')}: </span>
              <Chip>{front.kind}</Chip>
            </span>
          )
        }
      />
      <EntityBody bodies={d.bodies} />

      <Section title={t('module.paths')}>
        <SourceList items={(front.paths ?? []).map((p) => ({ path: p }))} />
      </Section>

      {front.endpoints && front.endpoints.length > 0 && (
        <Section title={t('module.endpoints')}>
          <div className="overflow-x-auto">
            <table className="table">
              <thead>
                <tr>
                  <th>{t('module.method')}</th>
                  <th>{t('module.path')}</th>
                  <th>{t('entity.roles')}</th>
                  <th>{t('module.note')}</th>
                </tr>
              </thead>
              <tbody>
                {front.endpoints.map((e) => (
                  <tr key={`${e.method} ${e.path}`}>
                    <td>
                      <Chip className="font-mono">{e.method}</Chip>
                    </td>
                    <td>
                      <code className="text-xs break-all">{e.path}</code>
                    </td>
                    <td>{e.roles?.length ? <RoleNames ids={e.roles} /> : '—'}</td>
                    <td>
                      <Loc v={e.note} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Section>
      )}

      <div className="grid gap-x-8 sm:grid-cols-2">
        <Section title={t('module.tables')}>
          {front.tables?.length ? (
            <div className="flex flex-wrap gap-1">
              {front.tables.map((x) => (
                <Chip key={x} className="font-mono">
                  {x}
                </Chip>
              ))}
            </div>
          ) : (
            <Muted>{t('entity.none')}</Muted>
          )}
        </Section>
        <Section title={t('module.dependsOn')}>
          <EntityList kind="module" ids={front.dependsOn} />
        </Section>
        <Section title={t('module.dependents')}>
          <EntityList kind="module" ids={refIds(d.refsTo, 'module', 'dependsOn')} />
        </Section>
        <Section title={t('module.screens')}>
          <EntityList kind="screen" ids={refIds(d.refsTo, 'screen')} />
        </Section>
        <Section title={t('module.features')}>
          <EntityList kind="feature" ids={refIds(d.refsTo, 'feature')} />
        </Section>
      </div>

      <Section title={t('entity.code')}>
        <SourceList items={d.sources} />
      </Section>
    </article>
  )
}
