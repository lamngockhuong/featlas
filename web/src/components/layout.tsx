import { Link, Outlet, useRouterState } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import { useProject } from '../api/client'
import { useLiveReload } from '../api/events'
import { isStatic } from '../api/url'
import { UI_LANGS, type UiLang } from '../lib/i18n'
import { useContentLang, usePrefs, useT } from '../lib/prefs'
import { KEYS, load, save } from '../lib/storage'
import { useCanReview } from './review-actions'
import { useReviewerIdentity } from './reviewer-identity'
import { SearchDialog } from './search-dialog'
import { Sidebar } from './sidebar'

export function Layout() {
  const t = useT()
  const prefs = usePrefs()
  const content = useContentLang()
  const { data: project } = useProject()
  const canReview = useCanReview()
  const identity = useReviewerIdentity()
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const [searchOpen, setSearchOpen] = useState(false)
  const [drawer, setDrawer] = useState(false)
  const [collapsed, setCollapsed] = useState(() => load(KEYS.sidebar) === 'collapsed')

  useLiveReload()

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault()
        setSearchOpen(true)
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])

  // Close the mobile drawer after navigating.
  // biome-ignore lint/correctness/useExhaustiveDependencies: runs on route change only
  useEffect(() => setDrawer(false), [pathname])

  useEffect(() => {
    document.documentElement.lang = prefs.uiLang
  }, [prefs.uiLang])

  useEffect(() => {
    if (project) document.title = `${project.config.project.name} · muninn`
  }, [project])

  const toggleNav = () => {
    if (window.matchMedia('(min-width: 1024px)').matches) {
      setCollapsed((c) => {
        save(KEYS.sidebar, c ? null : 'collapsed')
        return !c
      })
    } else setDrawer((d) => !d)
  }

  return (
    <div className="min-h-dvh">
      <a href="#main" className="sr-only focus:not-sr-only focus:absolute focus:z-50 focus:p-2">
        Skip to content
      </a>
      <header className="sticky top-0 z-30 flex h-14 items-center gap-2 border-b border-slate-200 bg-white/90 px-3 backdrop-blur dark:border-slate-800 dark:bg-slate-950/90">
        <button
          type="button"
          onClick={toggleNav}
          aria-label={t('nav.toggle')}
          aria-expanded={drawer || !collapsed}
          aria-controls="sidebar"
          className="icon-btn"
        >
          <svg viewBox="0 0 20 20" className="size-5" aria-hidden="true">
            <path d="M3 5h14M3 10h14M3 15h14" stroke="currentColor" strokeWidth="1.6" />
          </svg>
        </button>
        <Link to="/" className="truncate font-semibold">
          {project?.config.project.name ?? 'muninn'}
        </Link>
        {isStatic() && (
          <span className="hidden rounded bg-slate-100 px-1.5 py-0.5 text-xs text-slate-600 sm:inline dark:bg-slate-800 dark:text-slate-300">
            {t('app.static')}
          </span>
        )}
        <button
          type="button"
          onClick={() => setSearchOpen(true)}
          aria-label={t('header.search')}
          className="ml-auto flex items-center gap-2 rounded-md border border-slate-300 px-2 py-1 text-sm text-slate-500 hover:bg-slate-50 sm:w-56 dark:border-slate-700 dark:hover:bg-slate-900"
        >
          <svg viewBox="0 0 20 20" className="size-4" aria-hidden="true">
            <circle cx="9" cy="9" r="5.5" stroke="currentColor" strokeWidth="1.6" fill="none" />
            <path d="m13.5 13.5 3.5 3.5" stroke="currentColor" strokeWidth="1.6" />
          </svg>
          <span className="hidden sm:inline">{t('header.searchHint')}</span>
        </button>
        {content.available.length > 1 && (
          <select
            value={content.lang}
            onChange={(e) => prefs.setContentLang(e.target.value)}
            aria-label={t('header.contentLang')}
            title={t('header.contentLang')}
            className="input w-auto py-1 text-sm"
          >
            {content.available.map((l) => (
              <option key={l} value={l}>
                {l.toUpperCase()}
              </option>
            ))}
          </select>
        )}
        <select
          value={prefs.uiLang}
          onChange={(e) => prefs.setUiLang(e.target.value as UiLang)}
          aria-label={t('header.uiLang')}
          title={t('header.uiLang')}
          className="input hidden w-auto py-1 text-sm sm:block"
        >
          {UI_LANGS.map((l) => (
            <option key={l} value={l}>
              {{ en: 'English', vi: 'Tiếng Việt', ja: '日本語' }[l]}
            </option>
          ))}
        </select>
        {canReview && (
          <button
            type="button"
            onClick={identity.edit}
            title={t('header.reviewer')}
            aria-label={
              prefs.reviewer
                ? `${t('header.reviewer')}: ${prefs.reviewer}`
                : t('header.reviewerUnset')
            }
            className="hidden max-w-40 truncate rounded-md px-2 py-1 text-sm text-slate-600 hover:bg-slate-100 md:block dark:text-slate-300 dark:hover:bg-slate-800"
          >
            {prefs.reviewer ? `✎ ${prefs.reviewer}` : t('header.reviewerUnset')}
          </button>
        )}
        <button
          type="button"
          onClick={prefs.toggleTheme}
          aria-label={t('header.theme')}
          aria-pressed={prefs.theme === 'dark'}
          className="icon-btn"
        >
          {prefs.theme === 'dark' ? (
            <svg viewBox="0 0 20 20" className="size-5" aria-hidden="true">
              <circle cx="10" cy="10" r="4" fill="currentColor" />
              <path
                d="M10 1.5v2M10 16.5v2M1.5 10h2M16.5 10h2M4 4l1.4 1.4M14.6 14.6 16 16M4 16l1.4-1.4M14.6 5.4 16 4"
                stroke="currentColor"
                strokeWidth="1.5"
              />
            </svg>
          ) : (
            <svg viewBox="0 0 20 20" className="size-5" aria-hidden="true">
              <path d="M15.5 12.5A6.5 6.5 0 0 1 7.5 4.5a6.5 6.5 0 1 0 8 8Z" fill="currentColor" />
            </svg>
          )}
        </button>
      </header>

      <div className="flex">
        {drawer && (
          <button
            type="button"
            aria-label={t('nav.close')}
            onClick={() => setDrawer(false)}
            className="fixed inset-0 top-14 z-20 bg-slate-950/40 lg:hidden"
          />
        )}
        <aside
          id="sidebar"
          className={`fixed top-14 bottom-0 z-20 w-72 shrink-0 overflow-y-auto border-r border-slate-200 bg-white dark:border-slate-800 dark:bg-slate-950 lg:sticky lg:top-14 lg:h-[calc(100dvh-3.5rem)] ${
            drawer ? 'block' : 'hidden'
          } ${collapsed ? 'lg:hidden' : 'lg:block'}`}
        >
          <Sidebar />
          <div className="border-t border-slate-200 p-3 sm:hidden dark:border-slate-800">
            <label className="flex items-center justify-between gap-2 text-xs text-slate-500">
              {t('header.uiLang')}
              <select
                value={prefs.uiLang}
                onChange={(e) => prefs.setUiLang(e.target.value as UiLang)}
                className="input w-auto py-1 text-sm"
              >
                {UI_LANGS.map((l) => (
                  <option key={l} value={l}>
                    {{ en: 'English', vi: 'Tiếng Việt', ja: '日本語' }[l]}
                  </option>
                ))}
              </select>
            </label>
            {canReview && (
              <button type="button" onClick={identity.edit} className="btn mt-2 w-full">
                {prefs.reviewer
                  ? `${t('header.reviewer')}: ${prefs.reviewer}`
                  : t('header.reviewerUnset')}
              </button>
            )}
          </div>
        </aside>
        <main id="main" className="min-w-0 flex-1">
          <Outlet />
        </main>
      </div>
      {searchOpen && <SearchDialog onClose={() => setSearchOpen(false)} />}
    </div>
  )
}
