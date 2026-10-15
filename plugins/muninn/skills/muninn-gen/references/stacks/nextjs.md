# Next.js

- **App Router** (`app/`): each `page.tsx` is a screen; folders are segments (`[id]` dynamic, `(group)`
  ignored in the URL, `@slot` parallel routes). `layout.tsx` holds shared navigation and often auth checks.
- **Pages Router** (`pages/`): each file is a route; `pages/api/*` are endpoints.
- **Guards**: `middleware.ts` (matcher + redirect), `auth()`/`getServerSession()` checks in layouts or
  pages, NextAuth/Auth.js callbacks for roles.
- **Data**: Server Components fetching directly (ORM calls → `tables`), Server Actions (`'use server'`,
  form submissions), Route Handlers in `app/**/route.ts` (`export async function GET/POST`) → `api`.
- **Navigation**: `<Link href>`, `router.push`, `redirect()`.
