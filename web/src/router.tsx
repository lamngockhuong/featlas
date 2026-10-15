import type { QueryClient } from '@tanstack/react-query'
import {
  createBrowserHistory,
  createHashHistory,
  createRootRouteWithContext,
  createRoute,
  createRouter,
  type RouterHistory,
} from '@tanstack/react-router'
import { lazy, Suspense } from 'react'
import { REVIEW_STATES, type ReviewState } from './api/types'
import { isStatic } from './api/url'
import { Layout } from './components/layout'
import { Loading, NotFound } from './components/query-state'
import { FeaturePage } from './pages/feature-page'
import { FlowPage } from './pages/flow-page'
import { GlossaryPage } from './pages/glossary-page'
import { HomePage } from './pages/home-page'
import { ModulePage } from './pages/module-page'
import type { ReviewSearch } from './pages/review-page'
import { ReviewPage } from './pages/review-page'
import { ScreenPage } from './pages/screen-page'

// The graph library is large and used by one page only.
const GraphPage = lazy(() => import('./pages/graph-page'))

const str = (v: unknown): string | undefined => (typeof v === 'string' && v ? v : undefined)

const rootRoute = createRootRouteWithContext<{ queryClient: QueryClient }>()({
  component: Layout,
  notFoundComponent: NotFound,
})

const homeRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  validateSearch: (s: Record<string, unknown>): { role?: string } => ({ role: str(s.role) }),
  component: HomePage,
})

const featureRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/features/$id',
  component: FeaturePage,
})

const screenRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/screens/$id',
  validateSearch: (s: Record<string, unknown>): { v?: string } => ({ v: str(s.v) }),
  component: ScreenPage,
})

const flowRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/flows/$id',
  component: FlowPage,
})

const moduleRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/modules/$id',
  component: ModulePage,
})

const glossaryRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/glossary',
  component: GlossaryPage,
})

const graphRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/graph',
  component: () => (
    <Suspense fallback={<Loading />}>
      <GraphPage />
    </Suspense>
  ),
})

const reviewRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/review',
  validateSearch: (s: Record<string, unknown>): ReviewSearch => {
    const state = str(s.state)
    return {
      state:
        state === 'all' || (REVIEW_STATES as string[]).includes(state ?? '')
          ? (state as ReviewState | 'all')
          : undefined,
      area: str(s.area),
      owner: str(s.owner),
      kind: str(s.kind),
    }
  },
  component: ReviewPage,
})

export const routeTree = rootRoute.addChildren([
  homeRoute,
  featureRoute,
  screenRoute,
  flowRoute,
  moduleRoute,
  glossaryRoute,
  graphRoute,
  reviewRoute,
])

export function createAppRouter(queryClient: QueryClient, history?: RouterHistory) {
  return createRouter({
    routeTree,
    context: { queryClient },
    // The static export is opened from any folder or file:// URL, where only
    // the hash part of the URL is under our control.
    history: history ?? (isStatic() ? createHashHistory() : createBrowserHistory()),
    scrollRestoration: true,
    defaultPreload: false,
  })
}

declare module '@tanstack/react-router' {
  interface Register {
    router: ReturnType<typeof createAppRouter>
  }
}
