import { useParams } from '@tanstack/react-router'
import { useCallback, useEffect, useRef, useState } from 'react'
import { useEntity } from '../api/client'
import type { FlowFront, FlowStep, Media } from '../api/types'
import { RoleNames } from '../components/config-labels'
import { CopyButton } from '../components/copy-button'
import { EntityBody } from '../components/entity-body'
import { EntityHeader } from '../components/entity-header'
import { EntityLink } from '../components/entity-link'
import { useEntityTitle } from '../components/entity-refs'
import { Loc } from '../components/loc-text'
import { ErrorBox, Loading } from '../components/query-state'
import { Chip, Section } from '../components/section'
import { BoxHighlight, ShotFrame } from '../components/shot-frame'
import { SourceList } from '../components/source-list'
import { useContentLang, useT } from '../lib/prefs'

export function FlowPage() {
  const t = useT()
  const title = useEntityTitle()
  const { id } = useParams({ from: '/flows/$id' })
  const q = useEntity('flow', id)
  const [tour, setTour] = useState<number | null>(null)
  if (q.isPending) return <Loading />
  if (q.isError) return <ErrorBox error={q.error} retry={() => q.refetch()} />
  const d = q.data
  const front = d.front as FlowFront
  const steps: FlowStep[] = Array.isArray(front.steps) ? front.steps : []
  const mediaOf = (n: number) => d.steps?.find((s) => s.n === n)?.media

  return (
    <article className="mx-auto max-w-5xl px-4 py-6 sm:px-6">
      <EntityHeader
        entity={d}
        issues={d.issues}
        meta={
          <>
            {front.role && (
              <span>
                <span className="text-slate-500">{t('flow.role')}: </span>
                <RoleNames ids={[front.role]} />
              </span>
            )}
            {front.start && (
              <span>
                <span className="text-slate-500">{t('flow.start')}: </span>
                <EntityLink kind="screen" id={front.start} className="link">
                  {title('screen', front.start)}
                </EntityLink>
              </span>
            )}
            {front.end && (
              <span>
                <span className="text-slate-500">{t('flow.end')}: </span>
                <EntityLink kind="screen" id={front.end} className="link">
                  {title('screen', front.end)}
                </EntityLink>
              </span>
            )}
            {d.feature && (
              <span>
                <span className="text-slate-500">{t('entity.feature')}: </span>
                <EntityLink kind="feature" id={d.feature} className="link">
                  {title('feature', d.feature)}
                </EntityLink>
              </span>
            )}
          </>
        }
      />
      {front.goal && (
        <p className="mt-4 text-slate-700 dark:text-slate-300">
          <span className="font-medium">{t('flow.goal')}: </span>
          <Loc v={front.goal} />
        </p>
      )}

      <Section
        title={
          <span className="flex items-center justify-between gap-2">
            {t('flow.steps')}
            {steps.length > 0 && (
              <button type="button" className="btn btn-primary" onClick={() => setTour(0)}>
                {t('flow.tour')}
              </button>
            )}
          </span>
        }
      >
        <ol className="space-y-8">
          {steps.map((s) => (
            <li key={s.n} id={`step-${s.n}`}>
              <StepCard step={s} media={mediaOf(s.n)} />
            </li>
          ))}
        </ol>
      </Section>

      <EntityBody bodies={d.bodies} />

      <Section title={t('entity.code')}>
        <SourceList items={d.sources} />
      </Section>

      {tour !== null && steps.length > 0 && (
        <Tour steps={steps} mediaOf={mediaOf} index={tour} setIndex={setTour} />
      )}
    </article>
  )
}

function StepCard({ step, media }: { step: FlowStep; media: Media | undefined }) {
  const t = useT()
  const { text } = useContentLang()
  const hasImage = !step.run && (step.screen || step.shot)
  return (
    <div className="rounded-lg border border-slate-200 p-4 dark:border-slate-800">
      <div className="flex flex-wrap items-center gap-2">
        <span className="flex size-7 items-center justify-center rounded-full bg-slate-900 text-sm font-bold text-white dark:bg-slate-100 dark:text-slate-900">
          {step.n}
        </span>
        <Chip className="font-medium uppercase">{t(`action.${step.action}`) || step.action}</Chip>
        <h3 className="font-medium">
          <Loc v={step.label} />
        </h3>
        {step.screen && (
          <EntityLink kind="screen" id={step.screen} className="link ml-auto text-xs">
            {step.screen}
          </EntityLink>
        )}
      </div>
      {step.desc && (
        <p className="mt-2 text-sm text-slate-600 dark:text-slate-400">
          <Loc v={step.desc} />
        </p>
      )}
      {hasImage && (
        <div className="mt-3">
          <ShotFrame media={media} alt={text(step.label)}>
            {media && step.box && (
              <BoxHighlight box={step.box} media={media} label={text(step.label)} />
            )}
          </ShotFrame>
        </div>
      )}
      {step.run && (
        <div className="mt-3">
          {step.run.where && (
            <p className="mb-1 text-xs text-slate-500">
              {t('flow.where', { where: text(step.run.where) })}
            </p>
          )}
          <div className="flex items-start gap-2 rounded-md bg-slate-950 p-3 font-mono text-sm text-emerald-300">
            <pre className="min-w-0 flex-1 overflow-x-auto whitespace-pre-wrap">
              <span aria-hidden="true" className="text-slate-500 select-none">
                ${' '}
              </span>
              {step.run.cmd}
            </pre>
            <CopyButton text={step.run.cmd} />
          </div>
        </div>
      )}
      {step.sample && step.sample.length > 0 && (
        <div className="mt-3">
          <p className="mb-1 text-xs font-medium text-slate-500">{t('flow.sample')}</p>
          <ul className="divide-y divide-slate-200 rounded-md border border-slate-200 text-sm dark:divide-slate-800 dark:border-slate-800">
            {step.sample.map((s) => (
              <li key={s.label} className="flex flex-wrap items-center gap-x-3 gap-y-1 px-3 py-1.5">
                <span className="w-32 shrink-0 text-slate-500">{s.label}</span>
                <code className="rounded bg-slate-100 px-1.5 py-0.5 dark:bg-slate-800">
                  {s.value}
                </code>
                <CopyButton text={s.value} label={t('flow.copyValue', { label: s.label })} />
                {s.note && (
                  <span className="text-xs text-slate-500">
                    <Loc v={s.note} />
                  </span>
                )}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  )
}

function Tour({
  steps,
  mediaOf,
  index,
  setIndex,
}: {
  steps: FlowStep[]
  mediaOf: (n: number) => Media | undefined
  index: number
  setIndex: (i: number | null) => void
}) {
  const t = useT()
  const closeRef = useRef<HTMLButtonElement>(null)
  const last = steps.length - 1
  const go = useCallback(
    (delta: number) => setIndex(Math.min(last, Math.max(0, index + delta))),
    [index, last, setIndex],
  )
  useEffect(() => {
    closeRef.current?.focus()
  }, [])
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'ArrowRight') go(1)
      else if (e.key === 'ArrowLeft') go(-1)
      else if (e.key === 'Escape') setIndex(null)
      else return
      e.preventDefault()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [go, setIndex])
  const step = steps[index]
  if (!step) return null
  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label={t('flow.tour')}
      className="fixed inset-0 z-50 flex flex-col bg-white dark:bg-slate-950"
    >
      <div className="flex items-center gap-3 border-b border-slate-200 px-4 py-2 dark:border-slate-800">
        <span className="text-sm font-medium tabular-nums" aria-live="polite">
          {t('flow.of', { n: index + 1, total: steps.length })}
        </span>
        <span className="hidden text-xs text-slate-500 sm:inline">{t('flow.tourHelp')}</span>
        <div className="ml-auto flex gap-2">
          <button type="button" className="btn" onClick={() => go(-1)} disabled={index === 0}>
            ← <span className="sr-only sm:not-sr-only">{t('flow.prev')}</span>
          </button>
          <button type="button" className="btn" onClick={() => go(1)} disabled={index === last}>
            <span className="sr-only sm:not-sr-only">{t('flow.next')}</span> →
          </button>
          <button ref={closeRef} type="button" className="btn" onClick={() => setIndex(null)}>
            {t('flow.tourExit')}
          </button>
        </div>
      </div>
      <div className="flex-1 overflow-auto p-4">
        <div className="mx-auto max-w-5xl">
          <StepCard step={step} media={mediaOf(step.n)} />
        </div>
      </div>
    </div>
  )
}
