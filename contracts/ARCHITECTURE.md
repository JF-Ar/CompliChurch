# Igreja Organizada — Technical Architecture
Version: 1.0.0 · Stack locked for MVP

> **For Claude Code agents:** Read this file completely before writing any code.
> Do not invent endpoints, types, or conventions not listed here.
> The `contracts/openapi.yaml` file is the source of truth for all API contracts.

---

## 1. Repository Structure

```
/
├── backend/                    # Go service — Agent 1
│   ├── cmd/
│   │   └── api/
│   │       └── main.go         # Entry point
│   ├── internal/
│   │   ├── domain/             # Pure business entities and rules (no I/O)
│   │   │   ├── church/
│   │   │   ├── member/
│   │   │   ├── schedule/
│   │   │   ├── event/
│   │   │   ├── inventory/
│   │   │   ├── preaching/
│   │   │   └── notification/
│   │   ├── ports/              # Interfaces (what the domain needs from outside)
│   │   │   ├── repository.go   # DB port interfaces
│   │   │   ├── mailer.go       # Email port interface
│   │   │   └── storage.go      # File storage port interface
│   │   ├── adapters/           # Concrete implementations of ports
│   │   │   ├── postgres/       # sqlc-generated queries + repository impls
│   │   │   │   ├── queries/    # .sql files (input to sqlc)
│   │   │   │   ├── generated/  # sqlc output — DO NOT EDIT MANUALLY
│   │   │   │   └── *.go        # Repository adapter implementations
│   │   │   ├── resend/         # Email adapter (Resend.com)
│   │   │   └── r2/             # Cloudflare R2 storage adapter
│   │   ├── handlers/           # HTTP handlers (Chi router)
│   │   │   ├── auth.go
│   │   │   ├── churches.go
│   │   │   ├── members.go
│   │   │   ├── schedules.go
│   │   │   ├── events.go
│   │   │   ├── inventory.go
│   │   │   ├── preaching.go
│   │   │   └── middleware.go
│   │   └── services/           # Application services (orchestrate domain + ports)
│   │       ├── auth_service.go
│   │       ├── schedule_service.go
│   │       ├── event_service.go
│   │       └── inventory_service.go
│   ├── db/
│   │   └── migrations/         # .sql migration files (numbered)
│   │       └── 001_initial_schema.sql
│   ├── sqlc.yaml               # sqlc configuration
│   ├── Dockerfile
│   └── go.mod
│
├── frontend/                   # Next.js app — Agent 2
│   ├── app/                    # Next.js App Router
│   │   ├── (auth)/
│   │   │   ├── login/
│   │   │   └── register/
│   │   ├── (dashboard)/
│   │   │   ├── agenda/
│   │   │   ├── schedule/
│   │   │   ├── inventory/
│   │   │   └── members/
│   │   ├── layout.tsx
│   │   └── page.tsx
│   ├── components/
│   │   ├── ui/                 # Reusable primitives (shadcn/ui)
│   │   └── features/           # Feature-specific components
│   ├── lib/
│   │   ├── api.ts              # Typed API client (generated from openapi.yaml)
│   │   └── auth.ts             # Auth helpers
│   ├── Dockerfile
│   ├── next.config.ts
│   └── package.json
│
└── contracts/                  # Shared source of truth — read by both agents
    ├── openapi.yaml            # Full API specification
    └── ARCHITECTURE.md         # This file
```

---

## 2. Technology Decisions

### Backend

| Concern | Choice | Reason |
|---|---|---|
| Language | Go 1.22+ | Performance, simple concurrency, strong typing |
| Architecture | Hexagonal (ports & adapters) | Infra is swappable; domain has no I/O dependencies |
| Router | `chi` | Lightweight, idiomatic, middleware-friendly |
| SQL queries | `sqlc` | Type-safe, generated from SQL, zero reflection |
| DB driver | `pgx/v5` | Native PostgreSQL driver, best performance |
| Migrations | `golang-migrate` | Simple numbered `.sql` files, CLI-friendly |
| Auth | JWT (RS256) + refresh tokens | Stateless access token + revocable refresh token |
| JWT library | `golang-jwt/jwt/v5` | Standard, well-maintained |
| Password hash | `bcrypt` (cost 12) | Built into Go stdlib |
| Email | Resend.com via HTTP API | Simple REST API, generous free tier |
| File storage | Cloudflare R2 (S3-compatible) | Free tier 10GB, AWS SDK compatible |
| Config | `envconfig` | Struct-based env var parsing |
| Logging | `slog` (stdlib) | Structured JSON logs, no extra deps |
| Containerisation | Docker (single-stage for dev, multi-stage for prod) | |

### Frontend

| Concern | Choice | Reason |
|---|---|---|
| Framework | Next.js 14+ (App Router) | Best Claude Code performance; SSR + RSC |
| Language | TypeScript | Type safety aligned with OpenAPI types |
| Styling | Tailwind CSS | Claude Code generates Tailwind natively |
| Components | shadcn/ui | Accessible, unstyled base, Tailwind-friendly |
| Data fetching | TanStack Query (React Query) | Cache, loading states, mutations |
| Forms | React Hook Form + Zod | Validation aligned with API types |
| Auth state | `next-auth` or custom with HttpOnly cookie | Depends on complexity discovered during build |
| API client | Auto-generated from `openapi.yaml` (openapi-typescript) | Types shared from single source of truth |
| Calendar | `react-big-calendar` | Full-featured, customisable |

---

## 3. Authentication Flow

### Tokens

| Token | Type | TTL | Storage |
|---|---|---|---|
| Access token | JWT (RS256) | 15 minutes | Memory (JS variable) — never localStorage |
| Refresh token | JWT (RS256) | 30 days | HttpOnly cookie (Secure, SameSite=Strict) |

### Login sequence
```
Client                          Server
  │── POST /auth/login ────────►│
  │   { email, password }       │ 1. Verify password (bcrypt)
  │                             │ 2. Generate access_token (15m)
  │                             │ 3. Generate refresh_token (30d)
  │                             │ 4. Store refresh_token in DB (refresh_tokens table)
  │◄── 200 OK ─────────────────│
  │   { access_token,           │ Set-Cookie: refresh_token=...; HttpOnly; Secure
  │     member, church }        │
```

### Token refresh sequence
```
Client                          Server
  │── POST /auth/refresh ───────►│  (cookie sent automatically)
  │                              │ 1. Validate refresh_token signature
  │                              │ 2. Verify jti exists in DB and is not revoked
  │                              │ 3. Generate new access_token
  │                              │ 4. Rotate refresh_token (new jti, old one invalidated)
  │◄── 200 OK ──────────────────│
  │   { access_token }           │ Set-Cookie: refresh_token=... (rotated)
```

### Refresh tokens table (add to schema)
```sql
CREATE TABLE refresh_tokens (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id   UUID        NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    jti         UUID        NOT NULL UNIQUE,   -- JWT ID claim
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ,                   -- NULL = still valid
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_refresh_tokens_jti    ON refresh_tokens(jti);
CREATE INDEX idx_refresh_tokens_member ON refresh_tokens(member_id);
```

### Auth middleware
Every protected route extracts the JWT from `Authorization: Bearer <token>`, verifies the signature, and injects the following into the request context:

```go
type AuthContext struct {
    MemberID    uuid.UUID
    ChurchID    uuid.UUID   // primary church of the member
    BaseProfile string      // pastor | leadership | musician | member
    ChurchIDs   []uuid.UUID // all churches this member belongs to
}
```

**Every handler that touches domain data must filter by `ChurchID` from the auth context.** Never trust a `church_id` coming from the request body or query params for ownership checks.

---

## 4. API Conventions

### Base URL
```
/api/v1
```

### Request / Response format
- Content-Type: `application/json`
- Dates: ISO 8601 (`2025-04-22`)
- Timestamps: RFC 3339 with timezone (`2025-04-22T14:30:00Z`)
- IDs: UUID v4 strings

### Standard error envelope
```json
{
  "error": {
    "code": "SCHEDULE_ALREADY_EXISTS",
    "message": "A schedule for this Sunday already exists.",
    "field": "sunday_date"
  }
}
```

### Standard success envelope (lists)
```json
{
  "data": [...],
  "meta": {
    "total": 48,
    "page": 1,
    "per_page": 20
  }
}
```

### HTTP status codes used
| Code | When |
|---|---|
| 200 | Successful GET, PUT, PATCH |
| 201 | Successful POST (resource created) |
| 204 | Successful DELETE (no body) |
| 400 | Validation error (bad input) |
| 401 | Missing or invalid access token |
| 403 | Valid token but insufficient permissions |
| 404 | Resource not found (or not visible to this church) |
| 409 | Conflict (e.g. duplicate schedule for same Sunday) |
| 500 | Internal server error |

### Permissions shorthand used in endpoint table
- `P` = Pastor
- `L` = Leadership (and above)
- `M` = Musician (and above)
- `*` = Any authenticated member

---

## 5. API Endpoints

### Auth
| Method | Path | Access | Description |
|---|---|---|---|
| POST | `/auth/login` | public | Login with email + password |
| POST | `/auth/refresh` | cookie | Rotate refresh token, get new access token |
| POST | `/auth/logout` | `*` | Revoke current refresh token |
| POST | `/auth/logout-all` | `*` | Revoke all refresh tokens for this member |

### Churches
| Method | Path | Access | Description |
|---|---|---|---|
| GET | `/churches/me` | `*` | Get current member's primary church |
| PUT | `/churches/me` | `P` | Update church info |
| GET | `/churches/me/congregations` | `L` | List child congregations |
| POST | `/churches/me/congregations` | `P` | Add a congregation |

### Members
| Method | Path | Access | Description |
|---|---|---|---|
| GET | `/members` | `L` | List all members of current church |
| POST | `/members` | `L` | Create member + send invite email |
| POST | `/members/import` | `L` | Bulk import from CSV |
| GET | `/members/:id` | `L` | Get member detail |
| PUT | `/members/:id` | `L` | Update member |
| DELETE | `/members/:id` | `P` | Deactivate member (is_active = false) |
| GET | `/members/:id/roles` | `L` | List roles assigned to member |
| POST | `/members/:id/roles` | `L` | Assign role to member |
| DELETE | `/members/:id/roles/:role_id` | `L` | Remove role from member |
| GET | `/members/me` | `*` | Get own profile |
| PUT | `/members/me` | `*` | Update own profile |
| GET | `/members/me/instruments` | `*` | List own instruments |
| POST | `/members/me/instruments` | `*` | Add instrument to own profile |
| DELETE | `/members/me/instruments/:id` | `*` | Remove instrument from own profile |

### Roles
| Method | Path | Access | Description |
|---|---|---|---|
| GET | `/roles` | `L` | List all roles (system + church custom) |
| POST | `/roles` | `P` | Create custom role |
| PUT | `/roles/:id` | `P` | Update custom role (system roles are immutable) |
| DELETE | `/roles/:id` | `P` | Delete custom role |

### Agenda (Pastoral)
| Method | Path | Access | Description |
|---|---|---|---|
| GET | `/agenda/slots` | `*` | List pastor's available time slots |
| POST | `/agenda/slots` | `P` | Create availability slot |
| PUT | `/agenda/slots/:id` | `P` | Update availability slot |
| DELETE | `/agenda/slots/:id` | `P` | Remove availability slot |
| GET | `/agenda/events` | `*` | List events (filtered by role: members see confirmed only) |
| POST | `/agenda/events` | `*` | Request an appointment |
| GET | `/agenda/events/:id` | `*` | Get event detail |
| PUT | `/agenda/events/:id` | `P` | Update event (pastor edits directly) |
| POST | `/agenda/events/:id/confirm` | `P` | Confirm event → send GCal invite |
| POST | `/agenda/events/:id/decline` | `P` | Decline event → send email to requester |
| POST | `/agenda/events/:id/cancel` | `P` | Cancel confirmed event |
| GET | `/agenda/events/:id/attendees` | `L` | List attendees |
| POST | `/agenda/events/:id/attendees` | `P` | Add attendee |

### Schedules (Worship)
| Method | Path | Access | Description |
|---|---|---|---|
| GET | `/schedules` | `*` | List schedules (paginated, most recent first) |
| POST | `/schedules` | `L` | Create schedule (draft) |
| GET | `/schedules/:id` | `*` | Get schedule with all slots |
| PUT | `/schedules/:id` | `L` | Update schedule metadata |
| DELETE | `/schedules/:id` | `L` | Cancel schedule |
| POST | `/schedules/:id/publish` | `L` | Publish schedule → send email to all slots |
| GET | `/schedules/:id/slots` | `*` | List slots in this schedule |
| POST | `/schedules/:id/slots` | `L` | Add member to schedule |
| DELETE | `/schedules/:id/slots/:slot_id` | `L` | Remove member from schedule |
| POST | `/schedules/:id/slots/:slot_id/confirm` | `M` | Member confirms own slot |
| GET | `/schedules/suggest/:sunday_date` | `L` | Get auto-suggestion for a Sunday |
| GET | `/availability/exceptions` | `*` | List own unavailability (current month) |
| POST | `/availability/exceptions` | `*` | Mark a Sunday as unavailable |
| DELETE | `/availability/exceptions/:id` | `*` | Remove unavailability |
| GET | `/availability/exceptions/all` | `L` | List all exceptions for the church (for scheduling) |

### Instruments
| Method | Path | Access | Description |
|---|---|---|---|
| GET | `/instruments` | `*` | List all instruments (system + church custom) |
| POST | `/instruments` | `L` | Create custom instrument |
| DELETE | `/instruments/:id` | `L` | Delete custom instrument (system ones are immutable) |

### Inventory
| Method | Path | Access | Description |
|---|---|---|---|
| GET | `/inventory/categories` | `*` | List item categories |
| POST | `/inventory/categories` | `L` | Create category |
| PUT | `/inventory/categories/:id` | `L` | Update category |
| DELETE | `/inventory/categories/:id` | `L` | Delete category |
| GET | `/inventory/items` | `*` | List items (excludes soft-deleted by default) |
| POST | `/inventory/items` | `L` | Create item |
| GET | `/inventory/items/:id` | `*` | Get item detail |
| PUT | `/inventory/items/:id` | `L` | Update item |
| POST | `/inventory/items/:id/discard` | `L` | Soft-delete with reason=discarded |
| POST | `/inventory/items/:id/donate` | `L` | Soft-delete with reason=donated |
| POST | `/inventory/items/:id/photo` | `L` | Upload photo (multipart) → store in R2 |
| GET | `/inventory/loans` | `L` | List all loans for this church |
| POST | `/inventory/loans` | `*` | Request a loan |
| GET | `/inventory/loans/:id` | `L` | Get loan detail |
| POST | `/inventory/loans/:id/approve` | `L` | Approve loan request |
| POST | `/inventory/loans/:id/reject` | `L` | Reject loan request |
| POST | `/inventory/loans/:id/return` | `L` | Register return (with condition) |

### Notifications (internal, no frontend UI needed in MVP)
| Method | Path | Access | Description |
|---|---|---|---|
| GET | `/notifications` | `*` | List own notifications (last 30 days) |

### Preaching Schedule
| Method | Path | Access | Description |
|---|---|---|---|
| GET | `/preaching-schedules` | `*` | List schedules (filter by church_id and month) |
| POST | `/preaching-schedules` | `P` | Create schedule (draft) for a church + month |
| GET | `/preaching-schedules/:id` | `*` | Get schedule with all entries |
| PUT | `/preaching-schedules/:id` | `P` | Update schedule metadata |
| DELETE | `/preaching-schedules/:id` | `P` | Cancel schedule |
| POST | `/preaching-schedules/:id/publish` | `P` | Publish → email to all assigned members |
| GET | `/preaching-schedules/:id/entries` | `*` | List entries |
| POST | `/preaching-schedules/:id/entries` | `P` | Upsert entry for a Sunday |
| DELETE | `/preaching-schedules/:id/entries/:entry_id` | `P` | Remove entry |

---

## 6. Schedule Suggestion Algorithm

Endpoint: `GET /schedules/suggest/:sunday_date`

The service layer executes this logic — not the handler:

```
1. Fetch all members with base_profile = 'musician' in this church
2. Remove members who have an availability_exception for sunday_date
3. Remove members who were in a schedule_slot for the immediately
   preceding Sunday (consecutive Sunday rule)
   → Flag these members as "warning: consecutive" in the response
   → Do NOT exclude them — the leader decides
4. For remaining available members, group by their primary instrument
5. Return a suggested slot list: one member per instrument group
   (if multiple available for same instrument, pick the one with
   the fewest schedule_slots in the last 4 weeks)
6. Return alongside the suggestion:
   - available_members: full list of available musicians
   - warnings: members flagged for consecutive Sunday
   - unavailable: members with exceptions for this date
```

Response shape:
```json
{
  "sunday_date": "2025-05-04",
  "suggested_slots": [
    { "member_id": "...", "member_name": "João", "instrument_id": "...", "instrument_name": "Acoustic Guitar", "warning": null },
    { "member_id": "...", "member_name": "Maria", "instrument_id": "...", "instrument_name": "Lead Vocal", "warning": "consecutive_sunday" }
  ],
  "available_members": [...],
  "unavailable_members": [...]
}
```

---

## 6b. Preaching Schedule

**Model:** one `preaching_schedules` row per church per month. Each schedule has N `preaching_entries` rows — one per Sunday.

**Tables required (add to schema migration):**

```sql
CREATE TABLE preaching_schedules (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    church_id    UUID        NOT NULL REFERENCES churches(id) ON DELETE CASCADE,
    month        VARCHAR(7)  NOT NULL,  -- YYYY-MM
    status       VARCHAR(20) NOT NULL DEFAULT 'draft'
                     CHECK (status IN ('draft', 'published', 'cancelled')),
    created_by   UUID        NOT NULL REFERENCES members(id) ON DELETE RESTRICT,
    published_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_preaching_schedule UNIQUE (church_id, month)
);

CREATE TABLE preaching_entries (
    id                   UUID  PRIMARY KEY DEFAULT gen_random_uuid(),
    schedule_id          UUID  NOT NULL REFERENCES preaching_schedules(id) ON DELETE CASCADE,
    sunday_date          DATE  NOT NULL,
    direction_member_id  UUID  REFERENCES members(id) ON DELETE SET NULL,
    message_member_id    UUID  REFERENCES members(id) ON DELETE SET NULL,
    notes                TEXT,
    CONSTRAINT uq_preaching_entry UNIQUE (schedule_id, sunday_date)
);
```

**Business rules:**
- One schedule per church per month (`UNIQUE church_id + month`)
- One entry per Sunday per schedule (`UNIQUE schedule_id + sunday_date`)
- `direction_member_id` and `message_member_id` can be the same person
- Both fields are nullable — an entry can be saved with only one role filled
- Members from the matrix can be assigned to a congregation's schedule and vice-versa
- On publish: email sent to every unique member_id appearing in direction or message across all entries
- Schedule is visible to all authenticated members of the church hierarchy (matrix + congregations)
- Only pastor can create, edit and publish

---

All photo uploads go through the backend — the frontend never talks to R2 directly.

```
Frontend                    Backend                     Cloudflare R2
   │── POST /inventory/items/:id/photo ──►│
   │   multipart/form-data               │ 1. Validate file type (jpg, png, webp)
   │                                     │ 2. Validate size (max 5MB raw)
   │                                     │ 3. Resize + compress to max 800KB
   │                                     │ 4. Generate key: items/{church_id}/{item_id}.webp
   │                                     │──── PutObject ───────────────────────►│
   │                                     │◄─── OK ──────────────────────────────│
   │                                     │ 5. Update items.photo_url
   │◄── 200 { photo_url } ──────────────│
```

R2 bucket is private. Photo URLs returned by the API are pre-signed URLs valid for 1 hour, generated on each `GET /inventory/items` or `GET /inventory/items/:id` call.

---

## 8. Email Notifications

Email provider: **Resend.com** (free tier: 3,000 emails/month — sufficient for MVP pilot).

Emails are sent asynchronously: the handler returns 200 immediately, and a goroutine (or simple queue) sends the email and updates `notifications.status`.

| Template key | Trigger | Recipient |
|---|---|---|
| `member_welcome` | Member created | New member |
| `event_requested` | Member requests appointment | Pastor |
| `event_confirmed` | Pastor confirms | Requesting member |
| `event_declined` | Pastor declines | Requesting member |
| `schedule_published` | Schedule published | All members in slots |
| `availability_reminder` | 25th of each month (cron) | All musicians in church |
| `loan_requested` | Member requests loan | Asset manager |
| `loan_approved` | Loan approved | Requesting member |
| `loan_rejected` | Loan rejected | Requesting member |
| `loan_overdue` | Expected return date passed | Asset manager + borrower |
| `qty_min_alert` | Consumable below min qty | Asset manager |
| `preaching_schedule_published` | Preaching schedule published | All members assigned as direction or message |

---

## 9. Environment Variables

All services are configured exclusively via environment variables. No config files committed to the repo.

### Backend (`backend/.env.example`)
```env
# Database
DATABASE_URL=postgres://user:password@localhost:5432/igreja_organizada?sslmode=disable

# JWT — generate with: openssl genrsa -out private.pem 2048
JWT_PRIVATE_KEY_PATH=/secrets/jwt_private.pem
JWT_PUBLIC_KEY_PATH=/secrets/jwt_public.pem
JWT_ACCESS_TTL_MINUTES=15
JWT_REFRESH_TTL_DAYS=30

# Email (Resend)
RESEND_API_KEY=re_...
EMAIL_FROM=noreply@igreaorganizada.com.br

# Storage (Cloudflare R2)
R2_ACCOUNT_ID=...
R2_ACCESS_KEY_ID=...
R2_SECRET_ACCESS_KEY=...
R2_BUCKET_NAME=igreja-organizada
R2_PUBLIC_URL=https://pub-xxx.r2.dev  # or custom domain

# Server
PORT=8080
ENV=development  # development | production
LOG_LEVEL=info
```

### Frontend (`frontend/.env.example`)
```env
NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1
```

---

## 10. Docker Setup

### Backend `Dockerfile`
```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/api

# Run stage
FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
```

### Frontend `Dockerfile`
```dockerfile
FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:20-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
EXPOSE 3000
CMD ["node", "server.js"]
```

### `docker-compose.yml` (local dev)
```yaml
services:
  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: igreja_organizada
      POSTGRES_USER: dev
      POSTGRES_PASSWORD: dev
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
      - ./backend/db/migrations:/docker-entrypoint-initdb.d

  backend:
    build: ./backend
    ports:
      - "8080:8080"
    env_file: ./backend/.env
    depends_on:
      - db

  frontend:
    build: ./frontend
    ports:
      - "3000:3000"
    environment:
      NEXT_PUBLIC_API_URL: http://localhost:8080/api/v1
    depends_on:
      - backend

volumes:
  pgdata:
```

---

## 11. sqlc Configuration

File: `backend/sqlc.yaml`

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "internal/adapters/postgres/queries/"
    schema: "db/migrations/"
    gen:
      go:
        package: "db"
        out: "internal/adapters/postgres/generated"
        emit_json_tags: true
        emit_pointers_for_null_fields: true
        emit_enum_valid_method: true
```

---

## 12. Instructions for Claude Code Agents

### Agent 1 — Backend
```
You are building the Go backend for Igreja Organizada.
- Read /contracts/ARCHITECTURE.md fully before writing any code
- Read /contracts/openapi.yaml for all endpoint contracts
- Read /backend/db/migrations/001_initial_schema.sql for the database schema
- Architecture: hexagonal (ports & adapters)
- Router: chi
- Queries: sqlc (never write raw db calls outside adapters/postgres/)
- Every handler must extract church_id from JWT context, never from request body
- Return errors using the standard envelope defined in ARCHITECTURE.md section 4
- Write one handler file per domain (auth.go, members.go, schedules.go, etc.)
- Do not create endpoints not listed in ARCHITECTURE.md section 5
```

### Agent 2 — Frontend
```
You are building the Next.js frontend for Igreja Organizada.
- Read /contracts/ARCHITECTURE.md fully before writing any code
- Read /contracts/openapi.yaml for all API types and endpoints
- Framework: Next.js 14 App Router with TypeScript
- Styling: Tailwind CSS only — no inline styles, no CSS modules
- Components: shadcn/ui for primitives
- Data fetching: TanStack Query for all API calls
- Auth: access token in memory, refresh token handled by HttpOnly cookie automatically
- Never call an endpoint not listed in ARCHITECTURE.md section 5
- All API calls go through /frontend/lib/api.ts — never fetch() directly in components
```