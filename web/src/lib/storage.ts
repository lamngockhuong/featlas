// localStorage access that survives private mode and disabled storage.

export const KEYS = {
  theme: 'muninn.theme',
  uiLang: 'muninn.uiLang',
  contentLang: 'muninn.contentLang',
  reviewer: 'muninn.reviewer',
  token: 'muninn.token',
  sidebar: 'muninn.sidebar',
} as const

export function load(key: string): string | null {
  try {
    return localStorage.getItem(key)
  } catch {
    return null
  }
}

export function save(key: string, value: string | null): void {
  try {
    if (value === null) localStorage.removeItem(key)
    else localStorage.setItem(key, value)
  } catch {
    // Storage unavailable: the preference lasts for this page view only.
  }
}
