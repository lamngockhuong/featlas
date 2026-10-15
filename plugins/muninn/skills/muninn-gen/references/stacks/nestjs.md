# NestJS

- **Modules**: each `*.module.ts` is a good muninn module candidate (`kind: backend`, `paths` = its folder).
  `imports` give `dependsOn`.
- **Endpoints**: controllers (`@Controller('claims')`) with `@Get(':id')`, `@Post()`, ... A global prefix
  is set in `main.ts` (`app.setGlobalPrefix('api')`). Write the full path.
- **Roles**: `@UseGuards(AuthGuard, RolesGuard)` with `@Roles('approver')` or custom decorators; check the
  guard to see how roles are read. Put them in `endpoints[].roles`.
- **Validation and rules**: DTOs with class-validator decorators show required fields and formats;
  services hold the business rules and state transitions.
- **Data**: repositories/ORM calls in services → `tables`. Scheduled jobs (`@Cron`) and queue processors
  (`@Processor`) are `kind: job` modules.
