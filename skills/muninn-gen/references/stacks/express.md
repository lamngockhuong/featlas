# Express

- **Endpoints**: `app.get/post(...)` and `express.Router()` files mounted with `app.use('/api/claims',
  router)`. Concatenate mount paths with router paths.
- **Roles**: middleware in the route chain (`requireAuth`, `requireRole('approver')`, passport strategies).
  Middleware applied with `router.use` covers every route below it.
- **Validation**: zod/joi/express-validator schemas next to handlers show required fields and formats.
- **Data**: ORM or SQL calls in handlers or service files → `tables`. Background workers usually live in
  separate entry points (`worker.ts`, queue consumers) → `kind: job`.
