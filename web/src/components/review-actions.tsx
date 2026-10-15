import { useMutation, useQueryClient } from '@tanstack/react-query'
import { type FormEvent, useState } from 'react'
import { ApiError, postReview, useProject } from '../api/client'
import type { ReviewRequest, Summary } from '../api/types'
import { isStatic } from '../api/url'
import { useT } from '../lib/prefs'
import { useReviewerIdentity } from './reviewer-identity'

/** True when this client may record reviews. */
export function useCanReview(): boolean {
  const { data } = useProject()
  return !isStatic() && !!data?.capabilities.canReview && !data.capabilities.static
}

/** Approve / Request changes buttons for one entity. */
export function ReviewActions({
  entity,
}: {
  entity: Pick<Summary, 'kind' | 'id' | 'errors' | 'revision'>
}) {
  const t = useT()
  const qc = useQueryClient()
  const canReview = useCanReview()
  const identity = useReviewerIdentity()
  const [noteOpen, setNoteOpen] = useState(false)
  const [note, setNote] = useState('')
  const [noteError, setNoteError] = useState(false)

  const mutation = useMutation({
    mutationFn: async (req: Omit<ReviewRequest, 'by'>) => {
      const who = await identity.ensure()
      if (!who) return null
      // The revision ties the decision to the version on screen; the server
      // answers 409 when the files changed since.
      return postReview(
        entity.kind,
        entity.id,
        { ...req, by: who.by, revision: entity.revision },
        who.token || undefined,
      )
    },
    onSuccess: (detail) => {
      if (!detail) return
      qc.setQueryData(['entity', entity.kind, entity.id], detail)
      setNoteOpen(false)
      setNote('')
      void qc.invalidateQueries()
    },
    onError: (err) => {
      if (err instanceof ApiError && err.status === 401) identity.forgetToken()
      if (err instanceof ApiError && err.status === 409) void qc.invalidateQueries()
    },
  })

  if (!canReview) return null

  const blocked = entity.errors > 0
  const submitChanges = (e: FormEvent) => {
    e.preventDefault()
    if (!note.trim()) {
      setNoteError(true)
      return
    }
    mutation.mutate({ status: 'changes_requested', note: note.trim() })
  }
  const err = mutation.error

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap items-center gap-2">
        <button
          type="button"
          className="btn btn-approve"
          disabled={mutation.isPending || blocked}
          onClick={() => mutation.mutate({ status: 'approved', note: '' })}
        >
          {t('review.approve')}
        </button>
        <button
          type="button"
          className="btn"
          disabled={mutation.isPending || blocked}
          aria-expanded={noteOpen}
          onClick={() => {
            setNoteOpen((o) => !o)
            setNoteError(false)
          }}
        >
          {t('review.requestChanges')}
        </button>
        {mutation.isPending && (
          <span role="status" className="text-xs text-slate-500">
            {t('review.saving')}
          </span>
        )}
        {blocked && <span className="text-xs text-red-600">{t('review.fixFirst')}</span>}
      </div>
      {noteOpen && (
        <form onSubmit={submitChanges} className="flex max-w-xl flex-col gap-2">
          <label className="text-sm font-medium">
            {t('review.note')}
            <textarea
              value={note}
              onChange={(e) => {
                setNote(e.target.value)
                setNoteError(false)
              }}
              rows={3}
              placeholder={t('review.notePlaceholder')}
              aria-invalid={noteError}
              className="input mt-1 w-full"
            />
          </label>
          {noteError && <p className="text-xs text-red-600">{t('review.noteRequired')}</p>}
          <div className="flex gap-2">
            <button type="submit" className="btn btn-primary" disabled={mutation.isPending}>
              {t('review.send')}
            </button>
            <button type="button" className="btn" onClick={() => setNoteOpen(false)}>
              {t('review.cancel')}
            </button>
          </div>
        </form>
      )}
      {err && (
        <div role="alert" className="text-sm text-red-700 dark:text-red-300">
          {err.message}
          {err instanceof ApiError && err.issues.length > 0 && (
            <ul className="mt-1 list-disc pl-5 text-xs">
              {err.issues.map((i) => (
                <li key={`${i.file}:${i.path}:${i.message}`}>
                  {i.path && <code>{i.path}</code>} {i.message}
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  )
}

/** One line describing the last recorded review, with its note. */
export function LastReview({ review }: { review: Summary['review'] }) {
  const t = useT()
  if (!review.by || review.status === 'draft') return null
  const at = review.at ? new Date(review.at).toLocaleString() : ''
  return (
    <div className="text-xs text-slate-600 dark:text-slate-400">
      {t('review.lastBy', {
        status: t(review.status === 'approved' ? 'state.approved' : 'state.changes_requested'),
        by: review.by,
        at,
      })}
      {review.note && (
        <blockquote className="mt-1 border-l-2 border-slate-300 pl-2 whitespace-pre-wrap italic dark:border-slate-600">
          {review.note}
        </blockquote>
      )}
    </div>
  )
}
