# React Router (v6/v7)

- **Routes**: `createBrowserRouter([...])` or `<Routes><Route path element/>` trees; v7 framework mode uses
  `app/routes.ts` plus files in `app/routes/` (`claims.$id.tsx` = `/claims/:id`).
- **Route path**: write it as declared (`/claims/:id`).
- **Guards**: wrapper elements (`<RequireAuth>`, `<ProtectedRoute roles={[...]}>`) or `loader` functions that
  `redirect('/login')`. Collect the roles they check.
- **Data**: `loader`/`action` functions, or hooks with React Query/SWR inside the element. Follow them to the
  API client for `api`; `action` handlers reveal form submissions.
- **Navigation**: `<Link to>`, `<NavLink>`, `useNavigate()`, `redirect()` give `links` and hotspot `to`.
