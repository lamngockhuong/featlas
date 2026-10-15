import { useMemo } from 'react'
import { useProject } from '../api/client'
import type { Localized } from '../api/types'
import { useContentLang } from '../lib/prefs'

/** Role and area labels from muninn.yaml, keyed by id. */
export function useConfigLabels(): {
  roles: Map<string, Localized>
  areas: Map<string, Localized>
} {
  const { data } = useProject()
  return useMemo(
    () => ({
      roles: new Map((data?.config.roles ?? []).map((r) => [r.id, r.label])),
      areas: new Map((data?.config.areas ?? []).map((a) => [a.id, a.label])),
    }),
    [data],
  )
}

export function RoleNames({ ids }: { ids: string[] }) {
  const { text } = useContentLang()
  const { roles } = useConfigLabels()
  return <>{ids.map((r) => text(roles.get(r)) || r).join(', ')}</>
}
