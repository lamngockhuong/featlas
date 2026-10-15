export interface Picked {
  text: string
  /** Language actually shown when the requested one is missing. */
  fallback?: string
}

/**
 * Picks the text of a localized value ({en: ..., ja: ...}) for `lang`, falling
 * back to the default language, then to any language in a stable order.
 * Plain strings are language-neutral (labels, routes) and never fall back.
 */
export function pick(value: unknown, lang: string, def: string): Picked {
  if (typeof value === 'string') return { text: value }
  if (!value || typeof value !== 'object') return { text: '' }
  const map = value as Record<string, unknown>
  const has = (l: string) => typeof map[l] === 'string' && map[l] !== ''
  if (has(lang)) return { text: map[lang] as string }
  const order = [def, ...Object.keys(map).sort()]
  for (const l of order) {
    if (has(l)) return { text: map[l] as string, fallback: l }
  }
  return { text: '' }
}

/** Picks a Markdown body from the per-language bodies of an entity. */
export function pickBody(bodies: Record<string, string>, lang: string, def: string): Picked {
  return pick(bodies, lang, def)
}
