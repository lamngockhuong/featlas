import { createContext, type ReactNode, useCallback, useContext, useMemo, useState } from 'react'
import { useProject } from '../api/client'
import { detectUiLang, type Translate, translator, UI_LANGS, type UiLang } from './i18n'
import { type Picked, pick } from './localize'
import { KEYS, load, save } from './storage'

export type Theme = 'light' | 'dark'
export const REVIEWER_MAX = 100

interface Prefs {
  uiLang: UiLang
  setUiLang: (l: UiLang) => void
  contentLangPref: string | null
  setContentLang: (l: string) => void
  theme: Theme
  toggleTheme: () => void
  reviewer: string
  setReviewer: (name: string) => void
  token: string
  setToken: (token: string) => void
}

const PrefsContext = createContext<Prefs | null>(null)

function initialTheme(): Theme {
  if (typeof document !== 'undefined' && document.documentElement.classList.contains('dark'))
    return 'dark'
  return 'light'
}

function initialUiLang(): UiLang {
  const stored = load(KEYS.uiLang)
  return stored && (UI_LANGS as string[]).includes(stored) ? (stored as UiLang) : detectUiLang()
}

export function PrefsProvider({ children }: { children: ReactNode }) {
  const [uiLang, setUiLangState] = useState<UiLang>(initialUiLang)
  const [contentLangPref, setContentLangState] = useState<string | null>(() =>
    load(KEYS.contentLang),
  )
  const [theme, setTheme] = useState<Theme>(initialTheme)
  const [reviewer, setReviewerState] = useState(() => load(KEYS.reviewer) ?? '')
  const [token, setTokenState] = useState(() => load(KEYS.token) ?? '')

  const setUiLang = useCallback((l: UiLang) => {
    save(KEYS.uiLang, l)
    setUiLangState(l)
  }, [])
  const setContentLang = useCallback((l: string) => {
    save(KEYS.contentLang, l)
    setContentLangState(l)
  }, [])
  const toggleTheme = useCallback(() => {
    setTheme((prev) => {
      const next = prev === 'dark' ? 'light' : 'dark'
      document.documentElement.classList.toggle('dark', next === 'dark')
      save(KEYS.theme, next)
      return next
    })
  }, [])
  const setReviewer = useCallback((name: string) => {
    const clean = name.trim().slice(0, REVIEWER_MAX)
    save(KEYS.reviewer, clean || null)
    setReviewerState(clean)
  }, [])
  const setToken = useCallback((value: string) => {
    save(KEYS.token, value || null)
    setTokenState(value)
  }, [])

  const value = useMemo(
    () => ({
      uiLang,
      setUiLang,
      contentLangPref,
      setContentLang,
      theme,
      toggleTheme,
      reviewer,
      setReviewer,
      token,
      setToken,
    }),
    [
      uiLang,
      setUiLang,
      contentLangPref,
      setContentLang,
      theme,
      toggleTheme,
      reviewer,
      setReviewer,
      token,
      setToken,
    ],
  )
  return <PrefsContext.Provider value={value}>{children}</PrefsContext.Provider>
}

export function usePrefs(): Prefs {
  const ctx = useContext(PrefsContext)
  if (!ctx) throw new Error('usePrefs must be used inside PrefsProvider')
  return ctx
}

export function useT(): Translate {
  const { uiLang } = usePrefs()
  return useMemo(() => translator(uiLang), [uiLang])
}

export interface ContentLang {
  /** Language the reader chose, limited to languages.available. */
  lang: string
  /** Default content language of the project. */
  def: string
  available: string[]
  loc: (value: unknown) => Picked
  /** Shorthand for loc(value).text. */
  text: (value: unknown) => string
}

export function useContentLang(): ContentLang {
  const { contentLangPref } = usePrefs()
  const { data } = useProject()
  const def = data?.config.languages.default ?? 'en'
  const available = data?.config.languages.available?.length
    ? data.config.languages.available
    : [def]
  const lang = contentLangPref && available.includes(contentLangPref) ? contentLangPref : def
  return useMemo(
    () => ({
      lang,
      def,
      available,
      loc: (v: unknown) => pick(v, lang, def),
      text: (v: unknown) => pick(v, lang, def).text,
    }),
    [lang, def, available],
  )
}
