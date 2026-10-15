import { Link } from '@tanstack/react-router'
import { ApiError } from '../api/client'
import { useT } from '../lib/prefs'

export function Loading() {
  const t = useT()
  return (
    <p role="status" className="p-6 text-sm text-slate-500">
      {t('app.loading')}
    </p>
  )
}

export function NotFound() {
  const t = useT()
  return (
    <div className="mx-auto max-w-xl p-8 text-center">
      <h1 className="text-xl font-semibold">{t('app.notFound')}</h1>
      <p className="mt-2 text-slate-600 dark:text-slate-400">{t('app.notFoundBody')}</p>
      <Link to="/" className="mt-4 inline-block text-sky-700 underline dark:text-sky-400">
        {t('app.backHome')}
      </Link>
    </div>
  )
}

export function ErrorBox({ error, retry }: { error: unknown; retry?: () => void }) {
  const t = useT()
  if (error instanceof ApiError && error.status === 404) return <NotFound />
  const msg = error instanceof Error ? error.message : String(error)
  return (
    <div
      role="alert"
      className="m-6 rounded-md border border-red-300 bg-red-50 p-4 text-sm text-red-800 dark:border-red-800 dark:bg-red-950 dark:text-red-200"
    >
      <p className="font-medium">{t('app.error')}</p>
      <p className="mt-1 font-mono text-xs">{msg}</p>
      {retry && (
        <button
          type="button"
          onClick={retry}
          className="mt-3 rounded border border-red-400 px-2 py-1 text-xs"
        >
          {t('app.retry')}
        </button>
      )}
    </div>
  )
}
