import { queryOptions, useQuery } from '@tanstack/react-query'
import type {
  Detail,
  Graph,
  Issue,
  Kind,
  Project,
  ReviewRequest,
  SearchDoc,
  Summary,
  Term,
  Validation,
} from './types'
import { apiUrl, entityUrl, isStatic } from './url'

export class ApiError extends Error {
  constructor(
    readonly status: number,
    message: string,
    readonly issues: Issue[] = [],
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

async function readError(res: Response): Promise<ApiError> {
  let message = `${res.status} ${res.statusText}`.trim()
  let issues: Issue[] = []
  try {
    const body: unknown = await res.json()
    if (body && typeof body === 'object') {
      const b = body as { error?: unknown; issues?: unknown }
      if (typeof b.error === 'string') message = b.error
      if (Array.isArray(b.issues)) issues = b.issues as Issue[]
    }
  } catch {
    // Not JSON (proxy error page, static file missing): keep the status text.
  }
  return new ApiError(res.status, message, issues)
}

async function getJSON<T>(url: string): Promise<T> {
  let res: Response
  try {
    res = await fetch(url, { headers: { Accept: 'application/json' } })
  } catch (err) {
    throw new ApiError(0, err instanceof Error ? err.message : 'network error')
  }
  if (!res.ok) throw await readError(res)
  return (await res.json()) as T
}

export async function postReview(
  kind: Kind,
  id: string,
  body: ReviewRequest,
  token?: string,
): Promise<Detail> {
  if (isStatic()) throw new ApiError(403, 'static export is read-only')
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    Accept: 'application/json',
  }
  if (token) headers.Authorization = `Bearer ${token}`
  let res: Response
  try {
    res = await fetch(`${entityUrl(kind, id)}/review`, {
      method: 'POST',
      headers,
      body: JSON.stringify(body),
    })
  } catch (err) {
    throw new ApiError(0, err instanceof Error ? err.message : 'network error')
  }
  if (!res.ok) throw await readError(res)
  return (await res.json()) as Detail
}

// Query definitions. Keys are shared so a live reload event can invalidate
// everything at once while pages keep showing the previous data.

export const projectQuery = queryOptions({
  queryKey: ['project'],
  queryFn: () => getJSON<Project>(apiUrl('project')),
})

export const entitiesQuery = queryOptions({
  queryKey: ['entities'],
  queryFn: () => getJSON<Summary[]>(apiUrl('entities')),
})

export const entityQuery = (kind: Kind, id: string) =>
  queryOptions({
    queryKey: ['entity', kind, id],
    queryFn: () => getJSON<Detail>(entityUrl(kind, id)),
  })

export const searchIndexQuery = queryOptions({
  queryKey: ['search-index'],
  queryFn: () => getJSON<SearchDoc[]>(apiUrl('search-index')),
})

export const graphQuery = queryOptions({
  queryKey: ['graph'],
  queryFn: () => getJSON<Graph>(apiUrl('graph')),
})

export const glossaryQuery = queryOptions({
  queryKey: ['glossary'],
  queryFn: () => getJSON<Term[]>(apiUrl('glossary')),
})

export const validationQuery = queryOptions({
  queryKey: ['validation'],
  queryFn: () => getJSON<Validation>(apiUrl('validation')),
})

export const useProject = () => useQuery(projectQuery)
export const useEntities = () => useQuery(entitiesQuery)
export const useEntity = (kind: Kind, id: string) => useQuery(entityQuery(kind, id))
export const useGlossary = () => useQuery(glossaryQuery)
export const useValidation = () => useQuery(validationQuery)
