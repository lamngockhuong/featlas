# TanStack Router (React)

- **Routes**: file-based under `src/routes/` (`__root.tsx`, `index.tsx`, `claims/$id.tsx`, `_layout.tsx` for
  pathless layouts, `(group)` folders). The generated `routeTree.gen.ts` lists every route; read it first.
  Code-based setups call `createRoute({ getParentRoute, path })`.
- **Route path**: `createFileRoute('/claims/$id')`. Write `route: /claims/$id` in screen.md.
- **Guards**: `beforeLoad` in a route or layout (`throw redirect({ to: '/login' })`), often reading
  `context.auth`. Role checks usually sit in a parent layout such as `_authed.tsx` or `_admin.tsx`.
- **Data**: `loader` functions, `useSuspenseQuery`/`useQuery` with query options from `src/api` or
  `queries.ts`. Follow them to the HTTP client to fill `api`.
- **Search params**: `validateSearch` (often zod) shows filters kept in the URL; mention them in hotspots.
- **Navigation**: `<Link to="...">` and `navigate({ to })` give `links` and hotspot `to`.
