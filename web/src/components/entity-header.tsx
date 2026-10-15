import type { ReactNode } from 'react'
import type { Issue, Reason, Summary } from '../api/types'
import { useContentLang, useT } from '../lib/prefs'
import { Loc } from './loc-text'
import { LastReview, ReviewActions } from './review-actions'
import { StatusBadge } from './status-badge'

/** Why an approved entity needs a new review: edited content or changed sources. */
export function ReasonList({ reasons }: { reasons: Reason[] | undefined }) {
  const t = useT()
  if (!reasons?.length) return null
  return (
    <ul className="space-y-0.5 text-xs">
      {reasons.map((r) => (
        <li key={`${r.kind}:${r.path ?? ''}`} className="text-amber-800 dark:text-amber-300">
          {r.kind === 'content' ? (
            t('reason.content')
          ) : (
            <>
              <code className="break-all">{r.path}</code> {t(`source.${r.kind}`)}
            </>
          )}
        </li>
      ))}
    </ul>
  )
}

export function IssueList({ issues }: { issues: Issue[] }) {
  if (issues.length === 0) return null
  return (
    <ul className="space-y-1 text-sm">
      {issues.map((i) => (
        <li
          key={`${i.file}:${i.path ?? ''}:${i.message}`}
          className={
            i.severity === 'error'
              ? 'text-red-700 dark:text-red-300'
              : 'text-amber-800 dark:text-amber-300'
          }
        >
          <span className="font-medium uppercase text-[10px] tracking-wide">{i.severity}</span>{' '}
          <code className="text-xs">
            {i.file}
            {i.path ? ` ${i.path}` : ''}
          </code>{' '}
          {i.message}
        </li>
      ))}
    </ul>
  )
}

/** Title block shared by every entity page. */
export function EntityHeader({
  entity,
  issues = [],
  meta,
}: {
  entity: Summary
  issues?: Issue[]
  meta?: ReactNode
}) {
  const t = useT()
  const { loc } = useContentLang()
  const summary = entity.summary ? loc(entity.summary) : null
  return (
    <header className="border-b border-slate-200 pb-5 dark:border-slate-800">
      <p className="text-xs font-medium tracking-wide text-slate-500 uppercase">
        {t(`kind.${entity.kind}`)}
      </p>
      <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-2">
        <h1 className="text-2xl font-semibold">
          <Loc v={entity.title} fallback={entity.id} />
        </h1>
        <StatusBadge state={entity.status.state} />
      </div>
      {summary?.text && (
        <p className="mt-2 max-w-3xl text-slate-700 dark:text-slate-300">
          <Loc v={entity.summary} />
        </p>
      )}
      {meta && <div className="mt-3 flex flex-wrap gap-x-4 gap-y-1 text-sm">{meta}</div>}
      <div className="mt-4 flex flex-col gap-2">
        <LastReview review={entity.review} />
        <ReasonList reasons={entity.status.reasons} />
        <ReviewActions entity={entity} />
      </div>
      {issues.length > 0 && (
        <div className="mt-4 rounded-md border border-red-200 bg-red-50/60 p-3 dark:border-red-900 dark:bg-red-950/40">
          <p className="mb-1 text-sm font-medium">{t('entity.issues')}</p>
          <IssueList issues={issues} />
        </div>
      )}
    </header>
  )
}
