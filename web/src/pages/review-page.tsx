import { useNavigate, useSearch } from '@tanstack/react-router'
import { useEntities, useProject, useValidation } from '../api/client'
import { type Issue, KINDS, REVIEW_STATES, type ReviewState, type Summary } from '../api/types'
import { useConfigLabels } from '../components/config-labels'
import { IssueList, ReasonList } from '../components/entity-header'
import { EntityLink } from '../components/entity-link'
import { Loc } from '../components/loc-text'
import { ErrorBox, Loading } from '../components/query-state'
import { LastReview, ReviewActions, useCanReview } from '../components/review-actions'
import { Muted } from '../components/section'
import { StatusBadge } from '../components/status-badge'
import { useContentLang, useT } from '../lib/prefs'

export type ReviewSearch = {
  state?: ReviewState | 'all'
  area?: string
  owner?: string
  kind?: string
}

const NEEDS = 'needs'

export function ReviewPage() {
  const t = useT()
  const { text } = useContentLang()
  const search = useSearch({ from: '/review' })
  const navigate = useNavigate({ from: '/review' })
  const project = useProject()
  const entities = useEntities()
  const validation = useValidation()
  const canReview = useCanReview()
  const { areas } = useConfigLabels()

  if (project.isPending || entities.isPending) return <Loading />
  if (project.isError) return <ErrorBox error={project.error} retry={() => project.refetch()} />
  if (entities.isError) return <ErrorBox error={entities.error} retry={() => entities.refetch()} />

  const all = entities.data
  const owners = [...new Set(all.flatMap((e) => e.owners))].sort()
  const areaIds = [...new Set(all.map((e) => e.area).filter((a): a is string => !!a))]
  const state = search.state ?? NEEDS
  const rows = all
    .filter((e) => {
      if (state === NEEDS) {
        if (e.status.state === 'approved') return false
      } else if (state !== 'all' && e.status.state !== state) return false
      if (search.area && e.area !== search.area) return false
      if (search.owner && !e.owners.includes(search.owner)) return false
      if (search.kind && e.kind !== search.kind) return false
      return true
    })
    .sort(
      (a, b) =>
        REVIEW_STATES.indexOf(a.status.state) - REVIEW_STATES.indexOf(b.status.state) ||
        KINDS.indexOf(a.kind) - KINDS.indexOf(b.kind) ||
        a.id.localeCompare(b.id),
    )

  const setFilter = (patch: Partial<Record<keyof ReviewSearch, string>>) =>
    navigate({
      search: (prev) => {
        const next: Record<string, string | undefined> = { ...prev, ...patch }
        for (const k of Object.keys(next)) if (!next[k]) delete next[k]
        return next as ReviewSearch
      },
      replace: true,
      resetScroll: false,
    })

  const caps = project.data.capabilities
  const issues = validation.data?.issues ?? []
  const counts = project.data.counts

  return (
    <div className="mx-auto max-w-6xl px-4 py-6 sm:px-6">
      <h1 className="text-2xl font-semibold">{t('review.title')}</h1>
      {!canReview && (caps.readOnlyReason || caps.static) && (
        <p className="mt-2 rounded-md bg-amber-50 p-2 text-sm text-amber-900 dark:bg-amber-950 dark:text-amber-200">
          {t('review.readOnly', { reason: caps.readOnlyReason ?? t('app.static') })}
        </p>
      )}

      <details
        className="mt-4 rounded-md border border-slate-200 p-3 dark:border-slate-800"
        open={issues.some((i) => i.severity === 'error')}
      >
        <summary className="cursor-pointer text-sm font-medium">
          {t('review.validation')}:{' '}
          {validation.data
            ? `${t('review.errors', { n: validation.data.errors })}, ${t('review.warnings', { n: validation.data.warnings })}`
            : '…'}
        </summary>
        <div className="mt-2">
          {validation.isError ? (
            <ErrorBox error={validation.error} />
          ) : issues.length === 0 ? (
            <Muted>{t('review.validationOk')}</Muted>
          ) : (
            <IssueList issues={issues} />
          )}
        </div>
      </details>

      <div className="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-4">
        <Filter
          label={t('review.state')}
          value={state}
          onChange={(v) => setFilter({ state: v === NEEDS ? '' : v })}
          options={[
            [NEEDS, t('review.needsReview')],
            ['all', t('review.all')],
            ...REVIEW_STATES.map((s) => [s, `${t(`state.${s}`)} (${counts[s] ?? 0})`] as const),
          ]}
        />
        <Filter
          label={t('review.area')}
          value={search.area ?? ''}
          onChange={(v) => setFilter({ area: v })}
          options={[
            ['', t('review.all')],
            ...areaIds.map((a) => [a, text(areas.get(a)) || a] as const),
          ]}
        />
        <Filter
          label={t('review.owner')}
          value={search.owner ?? ''}
          onChange={(v) => setFilter({ owner: v })}
          options={[['', t('review.all')], ...owners.map((o) => [o, o] as const)]}
        />
        <Filter
          label={t('review.kind')}
          value={search.kind ?? ''}
          onChange={(v) => setFilter({ kind: v })}
          options={[['', t('review.all')], ...KINDS.map((k) => [k, t(`kinds.${k}`)] as const)]}
        />
      </div>

      {rows.length === 0 ? (
        <div className="mt-6">
          <Muted>{t('review.empty')}</Muted>
        </div>
      ) : (
        <ul className="mt-4 divide-y divide-slate-200 rounded-md border border-slate-200 dark:divide-slate-800 dark:border-slate-800">
          {rows.map((e) => (
            <ReviewRow key={e.key} e={e} issues={issues.filter((i) => i.entity === e.key)} />
          ))}
        </ul>
      )}
    </div>
  )
}

function ReviewRow({ e, issues }: { e: Summary; issues: Issue[] }) {
  const t = useT()
  return (
    <li className="flex flex-col gap-2 p-3 sm:p-4" data-testid="review-row" data-key={e.key}>
      <div className="flex flex-wrap items-center gap-2">
        <span className="text-xs font-medium tracking-wide text-slate-500 uppercase">
          {t(`kind.${e.kind}`)}
        </span>
        <EntityLink kind={e.kind} id={e.id} className="font-medium hover:underline">
          <Loc v={e.title} fallback={e.id} />
        </EntityLink>
        <StatusBadge state={e.status.state} />
        {e.errors > 0 && (
          <span className="text-xs text-red-700 dark:text-red-300">
            {t('review.errors', { n: e.errors })}
          </span>
        )}
        {e.warnings > 0 && (
          <span className="text-xs text-amber-700 dark:text-amber-300">
            {t('review.warnings', { n: e.warnings })}
          </span>
        )}
        {e.owners.length > 0 && (
          <span className="ml-auto text-xs text-slate-500">
            {t('entity.owners')}: {e.owners.join(', ')}
          </span>
        )}
      </div>
      <LastReview review={e.review} />
      {e.status.reasons && e.status.reasons.length > 0 && (
        <div>
          <p className="text-xs font-medium text-slate-500">{t('review.reasons')}</p>
          <ReasonList reasons={e.status.reasons} />
        </div>
      )}
      <IssueList issues={issues} />
      <ReviewActions entity={e} />
    </li>
  )
}

function Filter({
  label,
  value,
  onChange,
  options,
}: {
  label: string
  value: string
  onChange: (v: string) => void
  options: readonly (readonly [string, string])[]
}) {
  return (
    <label className="flex flex-col gap-1 text-xs font-medium text-slate-600 dark:text-slate-400">
      {label}
      <select value={value} onChange={(e) => onChange(e.target.value)} className="input text-sm">
        {options.map(([v, l]) => (
          <option key={v} value={v}>
            {l}
          </option>
        ))}
      </select>
    </label>
  )
}
