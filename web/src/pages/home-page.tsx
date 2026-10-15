import { Link, useNavigate, useSearch } from '@tanstack/react-router'
import { useEntities, useProject } from '../api/client'
import { REVIEW_STATES, type Summary } from '../api/types'
import { useConfigLabels } from '../components/config-labels'
import { Loc } from '../components/loc-text'
import { ErrorBox, Loading } from '../components/query-state'
import { Muted } from '../components/section'
import { ShotFrame } from '../components/shot-frame'
import { StatusBadge } from '../components/status-badge'
import { useContentLang, useT } from '../lib/prefs'

export function HomePage() {
  const t = useT()
  const { text } = useContentLang()
  const project = useProject()
  const entities = useEntities()
  const { role } = useSearch({ from: '/' })
  const navigate = useNavigate({ from: '/' })
  const { areas } = useConfigLabels()

  if (project.isPending || entities.isPending) return <Loading />
  if (project.isError) return <ErrorBox error={project.error} retry={() => project.refetch()} />
  if (entities.isError) return <ErrorBox error={entities.error} retry={() => entities.refetch()} />

  const p = project.data
  const roles = p.config.roles ?? []
  const features = entities.data.filter(
    (e) => e.kind === 'feature' && (!role || e.roles.includes(role)),
  )
  const areaOrder = [...areas.keys()]
  for (const f of features) {
    if (f.area && !areaOrder.includes(f.area)) areaOrder.push(f.area)
  }
  const groups = [...areaOrder, ''].map((a) => ({
    area: a,
    items: features.filter((f) => (f.area ?? '') === a),
  }))
  const pct = p.total ? Math.round((p.counts.approved / p.total) * 100) : 0

  return (
    <div className="mx-auto max-w-7xl px-4 py-6 sm:px-6">
      <header>
        <h1 className="text-2xl font-semibold">{p.config.project.name}</h1>
        {p.config.project.summary && (
          <p className="mt-1 max-w-3xl text-slate-600 dark:text-slate-400">
            <Loc v={p.config.project.summary} />
          </p>
        )}
      </header>

      <section
        aria-label={t('home.reviewed', { pct })}
        className="mt-5 rounded-lg border border-slate-200 p-4 dark:border-slate-800"
      >
        <div className="flex flex-wrap items-baseline justify-between gap-2">
          <p className="text-lg font-semibold">{t('home.reviewed', { pct })}</p>
          <p className="text-sm text-slate-500">
            {t('home.ofTotal', { n: p.counts.approved, total: p.total })}
          </p>
        </div>
        <div
          className="mt-2 flex h-2 overflow-hidden rounded-full bg-slate-100 dark:bg-slate-800"
          aria-hidden="true"
        >
          {REVIEW_STATES.map((s) =>
            p.counts[s] ? (
              <span
                key={s}
                className={`state-bar-${s}`}
                style={{ width: `${(p.counts[s] / p.total) * 100}%` }}
              />
            ) : null,
          )}
        </div>
        <ul className="mt-3 flex flex-wrap gap-2">
          {REVIEW_STATES.map((s) => (
            <li key={s}>
              <Link
                to="/review"
                search={{ state: s }}
                className="flex items-center gap-1.5 text-sm hover:underline"
              >
                <StatusBadge state={s} />
                <span className="tabular-nums">{p.counts[s] ?? 0}</span>
              </Link>
            </li>
          ))}
        </ul>
        {(p.validation.errors > 0 || p.validation.warnings > 0) && (
          <p className="mt-2 text-sm text-red-700 dark:text-red-300">
            <Link to="/review" className="hover:underline">
              {t('home.validation', p.validation)}
            </Link>
          </p>
        )}
      </section>

      {roles.length > 0 && (
        <fieldset className="mt-6 flex flex-wrap gap-2">
          <legend className="sr-only">{t('home.filterRole')}</legend>
          {[{ id: '', label: null }, ...roles].map((r) => {
            const on = (role ?? '') === r.id
            return (
              <button
                key={r.id || 'all'}
                type="button"
                aria-pressed={on}
                onClick={() => navigate({ search: { role: r.id || undefined }, replace: true })}
                className={`rounded-full px-3 py-1 text-sm ring-1 ${
                  on
                    ? 'bg-slate-900 text-white ring-slate-900 dark:bg-slate-100 dark:text-slate-900'
                    : 'ring-slate-300 hover:bg-slate-100 dark:ring-slate-700 dark:hover:bg-slate-800'
                }`}
              >
                {r.label ? text(r.label) || r.id : t('home.allRoles')}
              </button>
            )
          })}
        </fieldset>
      )}

      {features.length === 0 && (
        <div className="mt-6">
          <Muted>{t('home.noFeatures')}</Muted>
        </div>
      )}
      {groups
        .filter((g) => g.items.length > 0)
        .map((g) => (
          <section key={g.area || 'other'} className="mt-8">
            <h2 className="mb-3 text-lg font-semibold">
              {g.area ? text(areas.get(g.area)) || g.area : t('home.otherArea')}
            </h2>
            <ul className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {g.items.map((f) => (
                <li key={f.id}>
                  <FeatureCard f={f} />
                </li>
              ))}
            </ul>
          </section>
        ))}
    </div>
  )
}

function FeatureCard({ f }: { f: Summary }) {
  const t = useT()
  return (
    <Link
      to="/features/$id"
      params={{ id: f.id }}
      className="group flex h-full flex-col rounded-lg border border-slate-200 p-3 hover:border-sky-400 hover:shadow-sm dark:border-slate-800"
    >
      <ShotFrame media={f.thumb} alt="" />
      <div className="mt-3 flex items-start justify-between gap-2">
        <h3 className="font-semibold group-hover:underline">
          <Loc v={f.title} fallback={f.id} />
        </h3>
        <StatusBadge state={f.status.state} />
      </div>
      {f.summary && (
        <p className="mt-1 line-clamp-2 text-sm text-slate-600 dark:text-slate-400">
          <Loc v={f.summary} />
        </p>
      )}
      <p className="mt-auto pt-2 text-xs text-slate-500">
        {t('home.screens', { n: f.counts?.screens ?? 0 })} ·{' '}
        {t('home.flows', { n: f.counts?.flows ?? 0 })}
      </p>
    </Link>
  )
}
