import { useQueryClient } from '@tanstack/react-query'
import { useEffect } from 'react'
import { apiUrl, isStatic } from './url'

/**
 * Subscribes to the server's live reload stream. On every `changed` event all
 * queries are invalidated: mounted pages refetch in the background and keep
 * their current data on screen, so the scroll position does not move.
 */
export function useLiveReload(): void {
  const qc = useQueryClient()
  useEffect(() => {
    if (isStatic() || typeof EventSource === 'undefined') return
    const es = new EventSource(apiUrl('events'))
    const onChanged = () => {
      void qc.invalidateQueries()
    }
    es.addEventListener('changed', onChanged)
    return () => {
      es.removeEventListener('changed', onChanged)
      es.close()
    }
  }, [qc])
}
