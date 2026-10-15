import type { ReviewState } from '../api/types'
import { useT } from '../lib/prefs'

const STYLES: Record<ReviewState, string> = {
  draft:
    'bg-slate-100 text-slate-700 ring-slate-300 dark:bg-slate-800 dark:text-slate-200 dark:ring-slate-600',
  changes_requested:
    'bg-red-50 text-red-700 ring-red-300 dark:bg-red-950 dark:text-red-200 dark:ring-red-800',
  modified:
    'bg-violet-50 text-violet-700 ring-violet-300 dark:bg-violet-950 dark:text-violet-200 dark:ring-violet-800',
  outdated:
    'bg-amber-50 text-amber-800 ring-amber-300 dark:bg-amber-950 dark:text-amber-200 dark:ring-amber-800',
  approved:
    'bg-emerald-50 text-emerald-700 ring-emerald-300 dark:bg-emerald-950 dark:text-emerald-200 dark:ring-emerald-800',
}

const DOTS: Record<ReviewState, string> = {
  draft: 'bg-slate-400',
  changes_requested: 'bg-red-500',
  modified: 'bg-violet-500',
  outdated: 'bg-amber-500',
  approved: 'bg-emerald-500',
}

export function StatusBadge({ state, className = '' }: { state: ReviewState; className?: string }) {
  const t = useT()
  const help = t(`stateHelp.${state}`)
  return (
    <span
      data-testid="status-badge"
      data-state={state}
      title={help}
      className={`inline-flex shrink-0 items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium ring-1 ring-inset ${STYLES[state] ?? STYLES.draft} ${className}`}
    >
      <span aria-hidden="true" className={`size-1.5 rounded-full ${DOTS[state] ?? DOTS.draft}`} />
      {t(`state.${state}`)}
      <span className="sr-only">: {help}</span>
    </span>
  )
}

export function StateDot({ state }: { state: ReviewState }) {
  const t = useT()
  return (
    <span
      role="img"
      aria-label={t(`state.${state}`)}
      title={t(`stateHelp.${state}`)}
      className={`inline-block size-2 shrink-0 rounded-full ${DOTS[state] ?? DOTS.draft}`}
    />
  )
}
