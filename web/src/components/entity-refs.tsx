import { useMemo } from 'react'
import { useEntities } from '../api/client'
import type { Kind, Ref, Summary } from '../api/types'
import { useContentLang, useT } from '../lib/prefs'
import { EntityLink } from './entity-link'
import { Loc } from './loc-text'
import { StateDot } from './status-badge'

/** Entity summaries keyed by "kind/id". */
export function useEntityMap(): Map<string, Summary> {
  const { data } = useEntities()
  return useMemo(() => new Map((data ?? []).map((e) => [e.key, e])), [data])
}

/** Title of an entity in the content language, or its id while loading. */
export function useEntityTitle(): (kind: Kind, id: string) => string {
  const map = useEntityMap()
  const { text } = useContentLang()
  return (kind, id) => {
    const e = map.get(`${kind}/${id}`)
    return (e && text(e.title)) || id
  }
}

/** Unique ids of refsTo entries of one kind, in order. */
export function refIds(refs: Ref[], kind: Ref['kind'], field?: string): string[] {
  const out: string[] = []
  for (const r of refs) {
    if (r.kind === kind && (!field || r.field === field) && !out.includes(r.id)) out.push(r.id)
  }
  return out
}

/** A list of links to entities with their state dots. */
export function EntityList({ kind, ids }: { kind: Kind; ids: string[] | undefined }) {
  const t = useT()
  const map = useEntityMap()
  if (!ids?.length) return <p className="text-sm text-slate-500">{t('entity.none')}</p>
  return (
    <ul className="space-y-1 text-sm">
      {ids.map((id) => {
        const e = map.get(`${kind}/${id}`)
        return (
          <li key={id} className="flex items-center gap-2">
            {e && <StateDot state={e.status.state} />}
            <EntityLink
              kind={kind}
              id={id}
              className="text-sky-700 hover:underline dark:text-sky-400"
            >
              {e ? <Loc v={e.title} fallback={id} /> : id}
            </EntityLink>
          </li>
        )
      })}
    </ul>
  )
}
