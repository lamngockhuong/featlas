# Prisma

- **Tables**: `model Claim { ... }` in `prisma/schema.prisma`; the table name is `@@map("claims")` when
  present, else the model name. Use the real table name in `tables`.
- **Columns**: fields map to labels (`@map` gives column names). `enum` blocks list statuses.
- **Relations**: `@relation` fields show ownership (claim → user).
- **Queries**: `prisma.claim.findMany/create/update` in services and route handlers. `prisma/seed.ts` shows
  demo accounts and roles.
