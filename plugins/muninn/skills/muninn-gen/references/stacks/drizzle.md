# Drizzle ORM

- **Tables**: `pgTable('claims', { ... })` / `mysqlTable` / `sqliteTable` in `schema.ts` or `db/schema/*`.
  Use the SQL table name (first argument) in `tables`.
- **Columns**: map on-screen labels to `table.column` in screen `fields[].source`. Enums (`pgEnum`) list
  statuses for state diagrams.
- **Relations**: `relations(...)` and foreign keys (`.references(() => users.id)`) explain ownership rules.
- **Queries**: `db.select().from(claims)`, `db.query.claims.findMany(...)` in services show which screen
  reads which table. Migrations live in `drizzle/`; seed scripts reveal demo accounts and roles.
