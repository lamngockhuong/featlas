import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { type RouterHistory, RouterProvider } from '@tanstack/react-router'
import { useState } from 'react'
import { ReviewerProvider } from './components/reviewer-identity'
import { PrefsProvider } from './lib/prefs'
import { createAppRouter } from './router'

export function makeQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        // Data changes only when files change; the live reload stream
        // invalidates queries, so no polling or focus refetching is needed.
        staleTime: Number.POSITIVE_INFINITY,
        refetchOnWindowFocus: false,
        retry: (count, err) =>
          count < 2 && !(err instanceof Error && 'status' in err && err.status === 404),
      },
    },
  })
}

export function App({
  history,
  queryClient,
}: {
  history?: RouterHistory
  queryClient?: QueryClient
}) {
  const [qc] = useState(() => queryClient ?? makeQueryClient())
  const [router] = useState(() => createAppRouter(qc, history))
  return (
    <QueryClientProvider client={qc}>
      <PrefsProvider>
        <ReviewerProvider>
          <RouterProvider router={router} />
        </ReviewerProvider>
      </PrefsProvider>
    </QueryClientProvider>
  )
}
