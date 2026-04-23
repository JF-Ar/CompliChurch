# Igreja Organizada — Backend

## mandatory first steps
Before writing any code, read these two files completely:
- `/contracts/ARCHITECTURE.md` — stack decisions, folder structure, auth flow, API conventions, endpoint list, algorithms
- `/contracts/openapi.yaml` — all request/response schemas and endpoint contracts

Do not invent endpoints, types, field names or conventions not documented in those files.

## project
Church management SaaS MVP. Go backend, hexagonal architecture.

## stack
- Go 1.25
- Router: chi
- Queries: sqlc (never write raw db calls outside `internal/adapters/postgres/`)
- DB driver: pgx/v5
- Migrations: golang-migrate (files in `db/migrations/`)
- Auth: JWT RS256 + refresh tokens (see ARCHITECTURE.md section 3)
- Email: Resend.com via HTTP API
- File storage: Cloudflare R2 (S3-compatible, AWS SDK)
- Config: envconfig
- Logging: slog (stdlib, structured JSON)

## database
Schema is in `db/migrations/001_initial_schema.sql`.
PostgreSQL 15+. All PKs are UUID v4 via `gen_random_uuid()`.
Multi-tenant isolation via `church_id` on every domain table.

## folder structure
```
cmd/api/main.go
internal/
  domain/          pure business entities, no I/O
  ports/           interfaces (repository, mailer, storage)
  adapters/
    postgres/
      queries/     .sql files (sqlc input)
      generated/   sqlc output — never edit manually
    resend/
    r2/
  handlers/        one file per domain
  services/        application services
db/migrations/
sqlc.yaml
```

## auth rules
- Every protected handler extracts `church_id` from JWT context
- Never trust `church_id` from request body or query params for ownership checks
- Auth context struct is defined in ARCHITECTURE.md section 3

## api conventions
- Base path: `/api/v1`
- All responses use the standard envelope from ARCHITECTURE.md section 4
- Errors use `{ "error": { "code", "message", "field" } }`
- Lists use `{ "data": [...], "meta": { "total", "page", "per_page" } }`
- HTTP status codes follow the table in ARCHITECTURE.md section 4

## sqlc config
See `sqlc.yaml` in backend root. Generated code goes to `internal/adapters/postgres/generated/`. Never edit generated files.

## environment variables
All config via env vars. See ARCHITECTURE.md section 9 for full list.
Local dev uses `backend/.env` (not committed).

## do not
- Write endpoints not listed in ARCHITECTURE.md section 5
- Write raw SQL outside `internal/adapters/postgres/`
- Store secrets in code or config files
- Use `database/sql` directly — use pgx/v5 via sqlc