# Laravel

- **Routes**: `routes/web.php` (screens, Blade or Inertia pages) and `routes/api.php` (endpoints, prefixed
  `/api`). `php artisan route:list --json` gives the full table when you can run it.
- **Roles**: route middleware (`auth`, `can:approve,claim`, `role:approver` from spatie/permission),
  policies in `app/Policies`, gates in `AuthServiceProvider`.
- **Screens**: controllers return `view('claims.index')` (Blade in `resources/views`) or
  `Inertia::render('Claims/Index')` (components in `resources/js/Pages`).
- **Validation**: Form Requests (`app/Http/Requests`) list rules for each field.
- **Data**: Eloquent models (`protected $table`), migrations in `database/migrations`, seeders for demo
  accounts. Jobs in `app/Jobs`, scheduled tasks in `routes/console.php` or `Kernel.php`.
