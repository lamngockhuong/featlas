import {
  createContext,
  type FormEvent,
  type ReactNode,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react'
import { useProject } from '../api/client'
import { REVIEWER_MAX, usePrefs, useT } from '../lib/prefs'

export interface Identity {
  by: string
  token: string
}

interface Ctx {
  /** Resolves with the reviewer identity, asking for it first when unknown. */
  ensure: () => Promise<Identity | null>
  /** Opens the dialog to change the stored name. */
  edit: () => void
  /** Drops a rejected token so the next review asks again. */
  forgetToken: () => void
}

const IdentityContext = createContext<Ctx | null>(null)

export function ReviewerProvider({ children }: { children: ReactNode }) {
  const { reviewer, setReviewer, token, setToken } = usePrefs()
  const { data } = useProject()
  const needsToken = data?.capabilities.needsToken ?? false
  const [open, setOpen] = useState(false)
  const pending = useRef<((id: Identity | null) => void) | null>(null)

  const ensure = useCallback(() => {
    if (reviewer && (!needsToken || token)) return Promise.resolve({ by: reviewer, token })
    return new Promise<Identity | null>((resolve) => {
      pending.current?.(null)
      pending.current = resolve
      setOpen(true)
    })
  }, [reviewer, needsToken, token])

  const edit = useCallback(() => setOpen(true), [])
  const forgetToken = useCallback(() => setToken(''), [setToken])

  const close = (id: Identity | null) => {
    setOpen(false)
    pending.current?.(id)
    pending.current = null
  }

  const value = useMemo(() => ({ ensure, edit, forgetToken }), [ensure, edit, forgetToken])
  return (
    <IdentityContext.Provider value={value}>
      {children}
      {open && (
        <ReviewerDialog
          initialName={reviewer}
          askToken={needsToken}
          initialToken={token}
          onCancel={() => close(null)}
          onSave={(name, tok) => {
            setReviewer(name)
            if (needsToken) setToken(tok)
            close({ by: name.trim(), token: needsToken ? tok : token })
          }}
        />
      )}
    </IdentityContext.Provider>
  )
}

export function useReviewerIdentity(): Ctx {
  const ctx = useContext(IdentityContext)
  if (!ctx) throw new Error('useReviewerIdentity must be used inside ReviewerProvider')
  return ctx
}

function ReviewerDialog({
  initialName,
  askToken,
  initialToken,
  onCancel,
  onSave,
}: {
  initialName: string
  askToken: boolean
  initialToken: string
  onCancel: () => void
  onSave: (name: string, token: string) => void
}) {
  const t = useT()
  const [name, setName] = useState(initialName)
  const [tok, setTok] = useState(initialToken)
  const nameRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    nameRef.current?.focus()
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onCancel()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onCancel])

  const tooLong = name.trim().length > REVIEWER_MAX
  const valid = name.trim().length > 0 && !tooLong && (!askToken || tok.length > 0)
  const submit = (e: FormEvent) => {
    e.preventDefault()
    if (valid) onSave(name.trim(), tok)
  }
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-4">
      <form
        role="dialog"
        aria-modal="true"
        aria-labelledby="reviewer-title"
        onSubmit={submit}
        className="w-full max-w-sm rounded-lg bg-white p-5 shadow-xl dark:bg-slate-900"
      >
        <h2 id="reviewer-title" className="text-lg font-semibold">
          {t('reviewer.title')}
        </h2>
        <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">{t('reviewer.help')}</p>
        <label className="mt-4 block text-sm font-medium">
          {t('reviewer.name')}
          <input
            ref={nameRef}
            value={name}
            onChange={(e) => setName(e.target.value)}
            maxLength={REVIEWER_MAX}
            required
            autoComplete="name"
            className="input mt-1 w-full"
          />
        </label>
        {tooLong && <p className="mt-1 text-xs text-red-600">{t('reviewer.tooLong')}</p>}
        {askToken && (
          <label className="mt-3 block text-sm font-medium">
            {t('reviewer.token')}
            <input
              type="password"
              value={tok}
              onChange={(e) => setTok(e.target.value)}
              required
              autoComplete="current-password"
              className="input mt-1 w-full"
            />
            <span className="mt-1 block text-xs font-normal text-slate-500">
              {t('reviewer.tokenHelp')}
            </span>
          </label>
        )}
        <div className="mt-5 flex justify-end gap-2">
          <button type="button" onClick={onCancel} className="btn">
            {t('review.cancel')}
          </button>
          <button type="submit" disabled={!valid} className="btn btn-primary">
            {t('reviewer.save')}
          </button>
        </div>
      </form>
    </div>
  )
}
