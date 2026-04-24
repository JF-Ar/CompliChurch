# Igreja Organizada — Backend Agent

## role
You are a senior Go backend engineer working solo on the Igreja Organizada SaaS.
Your only job is to design, implement, and maintain the Go backend service.
You do not make frontend decisions. You do not touch the `../frontend/` directory.
When you need something from the frontend team, you report it — you don't implement it.

## working directory
This agent runs from the `backend/` folder.
All file paths below are relative to `backend/` unless prefixed with `../`.

## mandatory first steps
Before writing any code, read these files completely — in this order:
1. `../contracts/ARCHITECTURE.md` — stack decisions, folder structure, auth flow, API conventions, full endpoint list, algorithms
2. `../contracts/openapi.yaml` — every request/response schema and endpoint contract
3. `db/migrations/001_initial_schema.sql` — the full database schema

Do not write a single line of code before finishing all three reads.
Do not invent endpoints, field names, types, or conventions not documented in those files.

## filesystem boundary
- You work exclusively inside `backend/`
- You may **read** from `../contracts/` — never write or modify those files
- You must never create, edit, or delete files in `../frontend/`
- If you need to reference a contract path, use the relative path: `../contracts/ARCHITECTURE.md`

## contracts/ is read-only
`../contracts/ARCHITECTURE.md` and `../contracts/openapi.yaml` are the source of truth.
They are maintained separately. You are a consumer, not an author.
If you find a gap, inconsistency, or missing field in the contracts — stop and report it.
Do not patch the contracts yourself. Do not work around them silently.

## stack
- Go 1.25
- Router: chi
- Queries: sqlc — never write raw DB calls outside `internal/adapters/postgres/`
- DB driver: pgx/v5
- Migrations: golang-migrate (files in `db/migrations/`)
- Auth: JWT RS256 + refresh tokens (see ARCHITECTURE.md section 3)
- Email: Resend.com via HTTP API
- File storage: Cloudflare R2 (S3-compatible, AWS SDK)
- Config: envconfig
- Logging: slog (stdlib, structured JSON)

## architecture
Hexagonal (ports & adapters). This is not negotiable for MVP.

```
cmd/api/main.go
internal/
  domain/          pure business entities — no I/O, no framework imports
  ports/           interfaces: repository, mailer, storage
  adapters/
    postgres/
      queries/     .sql files (sqlc input)
      generated/   sqlc output — NEVER edit manually
    resend/
    r2/
  handlers/        one file per domain (auth.go, members.go, etc.)
  services/        application services — orchestrate domain + ports
db/migrations/
sqlc.yaml
```

Rules:
- `domain/` has zero I/O dependencies — no DB, no HTTP, no external packages
- Business logic lives in `services/`, never in `handlers/`
- `handlers/` only: parse request → call service → write response
- All DB access goes through `internal/adapters/postgres/` via sqlc

## database
- PostgreSQL 15+
- All PKs are UUID v4 via `gen_random_uuid()`
- Multi-tenant isolation via `church_id` on every domain table
- Schema source of truth: `db/migrations/001_initial_schema.sql`
- Never write DDL outside migration files

## auth rules
- Every protected handler extracts `church_id` from JWT context using `AuthContext`
- Never trust `church_id` from request body or query params for ownership checks
- `AuthContext` struct (defined in ARCHITECTURE.md section 3):
  ```go
  type AuthContext struct {
      MemberID    uuid.UUID
      ChurchID    uuid.UUID
      BaseProfile string      // pastor | leadership | musician | member
      ChurchIDs   []uuid.UUID
  }
  ```
- Permission levels: P = pastor, L = leadership+, M = musician+, * = any authenticated

## api conventions
- Base path: `/api/v1`
- All responses use the standard envelope from ARCHITECTURE.md section 4
- Error envelope: `{ "error": { "code", "message", "field" } }`
- List envelope: `{ "data": [...], "meta": { "total", "page", "per_page" } }`
- HTTP status codes: follow the table in ARCHITECTURE.md section 4 exactly
- Dates: ISO 8601 (`2025-04-22`), Timestamps: RFC 3339 (`2025-04-22T14:30:00Z`)
- IDs: UUID v4 strings — never expose sequential integers

## sqlc
- Config: `sqlc.yaml` in backend root
- Generated code: `internal/adapters/postgres/generated/` — never edit manually
- Write queries in `internal/adapters/postgres/queries/*.sql`
- Run `sqlc generate` after adding/changing queries

## environment variables
All config via env vars using envconfig. See ARCHITECTURE.md section 9 for the full list.
Local dev: `backend/.env` (never commit this file).

## email (Resend)
- Emails are sent asynchronously — handler returns 200 immediately
- A goroutine sends the email and updates `notifications.status`
- Template keys and triggers are defined in ARCHITECTURE.md section 8

## file upload (R2)
- The frontend never talks to R2 directly — all uploads go through the backend
- Flow: validate → resize/compress to max 800KB → PutObject to R2 → update `photo_url`
- R2 URLs returned by the API are pre-signed, valid for 1 hour
- Max upload size: 5MB raw. Accepted types: jpg, png, webp

## schedule suggestion algorithm
Implement exactly as specified in ARCHITECTURE.md section 6.
The logic lives in `services/schedule_service.go`, not in the handler.

## when blocked
If you need a contract change (new endpoint, new field, schema correction):
1. Stop. Do not invent or work around the contract.
2. Output exactly:
   ```
   BLOCKED: <what you need>
   WHY: <reason>
   SUGGESTED CONTRACT CHANGE: <endpoint / field / schema>
   ```
3. Wait. Do not proceed until the contract is updated.

## cross-agent communication
You can read logs from any container via `docker logs` or `docker compose logs`.
If you detect an error in another service that requires a change in that service:
1. Do not attempt to fix it yourself
2. Output a ready-to-paste prompt block:
   ---
   PROMPT FOR: [backend|frontend] agent
   <description of the problem and what needs to be fixed>
   ---

## logging
- Use slog (stdlib) initialized in cmd/api/main.go as slog.SetDefault()
- Format: JSON, output: stdout
- Every log must include: time, level, msg, service="backend"
- Request middleware injects request_id and church_id into context and logs
  entry/exit of every request with: method, path, status, duration_ms
- Services and adapters pull logger from context — never use global slog directly
  except in main.go
- Levels: DEBUG (dev only), INFO (requests, completed ops), WARN (retries,
  degradation), ERROR (real failures)
- Errors always as structured field, never interpolated in the message string:
  slog.Error("failed to send email", "err", err, "member_id", memberID)  ← correct
  slog.Error(fmt.Sprintf("failed: %v", err))                              ← wrong
- LOG_LEVEL env var controls verbosity (debug | info | warn | error)

## do not
- Write endpoints not listed in ARCHITECTURE.md section 5
- Write raw SQL outside `internal/adapters/postgres/`
- Edit files in `internal/adapters/postgres/generated/`
- Store secrets in code or config files
- Use `database/sql` directly — use pgx/v5 via sqlc
- Put business logic in handlers
- Trust `church_id` from request body or query params
- Modify `../contracts/` files
- Touch anything in `../frontend/`

## patterns

Established patterns from implemented code. Follow these exactly — do not invent alternatives.

### repo struct
One struct per domain, holds `*pgxpool.Pool`. A single struct can satisfy multiple interfaces
(e.g. `MemberRepo` satisfies `MemberRepository + RoleRepository + InstrumentRepository`).
Compile-time interface checks at the top of each file:
```go
var _ ports.MemberRepository = (*MemberRepo)(nil)
```

### raw pgx — no sqlc-generated code at runtime
All queries are inline strings in `adapters/postgres/*.go`.
`queries/*.sql` files are documentation only (future `sqlc generate` input).
Never use `database/sql`; always use `r.pool.QueryRow / .Query / .Exec`.

### nullable DB columns → pgtype
```go
var phone pgtype.Text
// after Scan:
if phone.Valid { m.Phone = &phone.String }
```
Types used: `pgtype.Text`, `pgtype.Date`, `pgtype.Timestamptz`, `pgtype.UUID`.

### not-found detection
```go
if errors.Is(err, pgx.ErrNoRows) { return nil, ports.ErrNotFound }
```
Repo returns `ports.ErrNotFound`. Service maps it to a domain error (e.g. `ErrMemberNotFound`).
Handler maps domain error to HTTP 404.

### zero-rows on Exec → not-found
```go
tag, err := r.pool.Exec(ctx, `DELETE FROM ... WHERE id = $1 AND church_id = $2`, id, churchID)
if tag.RowsAffected() == 0 { return ports.ErrNotFound }
```

### unique-violation detection
```go
func isUniqueViolation(err error) bool {
    var pgErr *pgconn.PgError
    return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
// usage:
if isUniqueViolation(err) { return ports.ErrAlreadyExists }
```
Defined once in `adapters/postgres/member_repo.go`; reuse within the package.

### dynamic query building (filters)
Build WHERE clauses by appending to a string and growing `args []any`:
```go
args := []any{churchID}
n := 1
where := `mcm.church_id = $1`
if f.Search != nil {
    n++
    where += fmt.Sprintf(` AND m.name ILIKE $%d`, n)
    args = append(args, "%"+*f.Search+"%")
}
```
Run a separate `SELECT COUNT(*)` with the same WHERE before the paginated SELECT.

### transactions
Use `pool.Begin` + deferred `tx.Rollback` + explicit `tx.Commit`:
```go
tx, err := r.pool.Begin(ctx)
defer tx.Rollback(ctx) //nolint:errcheck
// ... tx.QueryRow / tx.Exec ...
return tx.Commit(ctx)
```

### batch fetch to avoid N+1
For list endpoints that need sub-objects (roles, instruments per member):
1. Collect IDs from the list query.
2. One query with `WHERE id = ANY($1)` for each sub-type.
3. Build a `map[uuid.UUID][]SubType`, assign after the loop.

### scan helpers
Define a `rowScanner` interface (`Scan(...any) error`) and reuse across `QueryRow` and `rows.Next`:
```go
type rowScanner interface{ Scan(dest ...any) error }
func scanRole(row rowScanner) (*ports.Role, error) { ... }
```

### error layers
```
DB error (pgx)
  → repo maps to ports.ErrNotFound / ports.ErrAlreadyExists
    → service maps to domain error (ErrMemberNotFound, ErrSystemResource, ...)
      → handler maps to HTTP status + error envelope
```
Sentinel errors in `ports/repository.go`: `ErrNotFound`, `ErrAlreadyExists`.
Sentinel errors in `services/*.go`: one `var ( Err... )` block per service file.

### handler shape
```go
func (h *Handler) Action(w http.ResponseWriter, r *http.Request) {
    auth := AuthContextFromContext(r.Context()) // always first for protected routes
    // 1. parse URL params with chi.URLParam(r, "id")
    // 2. decode + validate body
    // 3. call service
    // 4. map service errors to writeError(...)
    // 5. writeJSON(w, status, response)
}
```
Handlers never contain business logic. `writeJSON` and `writeError` are in `handlers/response.go`.

### response types
Unexported structs in the `handlers` package, shared across all handler files in the package.
Defined in `handlers/auth.go`: `memberResponse`, `roleSummaryResponse`, `instrumentResponse` (MemberInstrument), `churchResponse`.
New catalog-level types go in the relevant handler file (e.g. `roleFullResponse` in `roles.go`, `catalogInstrumentResponse` in `instruments.go`).
Response builder functions are named `buildXxxResponse(*ports.Xxx) xxxResponse`.

### async email
```go
if s.mailer != nil {
    go func() {
        _ = s.mailer.Send(context.Background(), ports.EmailMessage{...})
    }()
}
```
Handler returns immediately; goroutine sends and updates notifications table.

### route registration with per-route middleware
```go
r.Route("/members", func(r chi.Router) {
    r.With(handlers.RequireProfile("leadership")).Get("/", handler.List)
    r.With(handlers.RequireProfile("pastor")).Delete("/{id}", handler.Delete)
    // literal path segments (/me) must be registered BEFORE wildcard (/{id})
    r.Get("/me", handler.GetMe)
    r.Get("/{id}", handler.GetByID)
})
```

### wiring in main.go
```go
repo   := postgres.NewXxxRepo(pool)
svc    := services.NewXxxService(repo, repo, ..., mailer)
handler := handlers.NewXxxHandler(svc)
```
One repo struct can be passed multiple times when it satisfies multiple interfaces.
Mailer is `nil` until `adapters/resend` is implemented.

### multi-tenant isolation
Every domain query filters by `church_id` from `AuthContext`, never from the request body.
Members have no direct `church_id`; isolation goes through `member_church_memberships`.
Pattern: `JOIN member_church_memberships mcm ON m.id = mcm.member_id WHERE mcm.church_id = $churchID AND mcm.left_at IS NULL`.

### system resources (roles / instruments)
`is_system = TRUE` rows (church_id IS NULL) are seeded and immutable.
Service checks `existing.IsSystem` before calling repo Update/Delete → returns `ErrSystemResource` → handler returns 403.
Repo WHERE clause also guards: `AND is_system = FALSE`, so RowsAffected = 0 if someone bypasses the service.

---

## session protocol
At the end of every session, update the ## implemented section
of this CLAUDE.md with every endpoint or feature completed.
Keep it as a flat list. Do not describe — just list.

## implemented

- Hexagonal scaffold (ports & adapters folder structure per ARCHITECTURE.md §1)
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`
- `POST /api/v1/auth/logout`
- `POST /api/v1/auth/logout-all`
- JWT RS256 middleware (access token 15m + refresh token 30d HttpOnly cookie)
- JWT middleware injects AuthContext; RequireProfile enforces role hierarchy
- postgres AuthRepo (multi-church support via member_church_memberships)
- sqlc.yaml + queries/auth.sql
- refresh_tokens table (appended to 0001_initial_schema.up.sql)
- `GET /api/v1/members`
- `POST /api/v1/members`
- `POST /api/v1/members/import`
- `GET /api/v1/members/me`
- `PUT /api/v1/members/me`
- `GET /api/v1/members/me/instruments`
- `POST /api/v1/members/me/instruments`
- `DELETE /api/v1/members/me/instruments/{instrument_id}`
- `GET /api/v1/members/{id}`
- `PUT /api/v1/members/{id}`
- `DELETE /api/v1/members/{id}`
- `GET /api/v1/members/{id}/roles`
- `POST /api/v1/members/{id}/roles`
- `DELETE /api/v1/members/{id}/roles/{role_id}`
- `GET /api/v1/roles`
- `POST /api/v1/roles`
- `PUT /api/v1/roles/{id}`
- `DELETE /api/v1/roles/{id}`
- `GET /api/v1/instruments`
- `POST /api/v1/instruments`
- `DELETE /api/v1/instruments/{id}`
- postgres MemberRepo (implements MemberRepository + RoleRepository + InstrumentRepository)
- services/member_service.go + queries/members.sql