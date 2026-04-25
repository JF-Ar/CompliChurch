# Igreja Organizada

SaaS for church management — worship scheduling, member management, pastoral agenda, and inventory control.

---

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Tech Stack](#tech-stack)
- [Prerequisites](#prerequisites)
- [Quick Start — Docker (recommended)](#quick-start--docker-recommended)
- [Development with Hot Reload](#development-with-hot-reload)
- [Running Locally Without Docker](#running-locally-without-docker)
- [Environment Variables](#environment-variables)
- [JWT Keys](#jwt-keys)
- [Database Migrations](#database-migrations)
- [API Reference](#api-reference)
- [Project Structure](#project-structure)

---

## Overview

| Service   | URL (local)                    | Description                      |
|-----------|--------------------------------|----------------------------------|
| Backend   | http://localhost:8080/api/v1   | Go REST API                      |
| Frontend  | http://localhost:3000          | Next.js web app                  |
| Database  | localhost:5432                 | PostgreSQL 16                    |

---

## Architecture

Hexagonal (ports & adapters) on the backend. The `contracts/` folder is the shared source of truth for both services.

```
/
├── backend/        Go service — hexagonal architecture (chi + pgx + sqlc)
├── frontend/       Next.js app — App Router + TanStack Query + shadcn/ui
└── contracts/
    ├── openapi.yaml          API contract (single source of truth)
    ├── ARCHITECTURE.md       Stack decisions, endpoint list, algorithms
    └── UI_UX_STANDARDS.md    Frontend quality bar
```

Multi-tenant: every domain table is scoped by `church_id`. The `church_id` is always extracted from the JWT — never from request body or query params.

---

## Tech Stack

### Backend

| Concern        | Choice                          |
|----------------|---------------------------------|
| Language       | Go 1.25                         |
| Router         | chi v5                          |
| Database       | PostgreSQL 16                   |
| DB driver      | pgx/v5                          |
| Queries        | sqlc (type-safe, generated)     |
| Migrations     | golang-migrate                  |
| Auth           | JWT RS256 + HttpOnly refresh token |
| Password hash  | bcrypt (cost 12)                |
| Email          | Resend.com (async)              |
| File storage   | Cloudflare R2 (S3-compatible)   |
| Config         | envconfig (struct-based)        |
| Logging        | slog (stdlib, structured JSON)  |

### Frontend

| Concern        | Choice                          |
|----------------|---------------------------------|
| Framework      | Next.js 16 / App Router         |
| Language       | TypeScript (strict)             |
| Styling        | Tailwind CSS v4                 |
| Components     | shadcn/ui (Radix primitives)    |
| Data fetching  | TanStack Query v5               |
| Forms          | React Hook Form + Zod           |
| Toasts         | Sonner                          |

---

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) + Docker Compose v2
- `openssl` (to generate JWT keys — one-time setup)
- Go 1.25+ (only for running backend locally without Docker)
- Node.js 20+ and npm (only for running frontend locally without Docker)

---

## Quick Start — Docker (recommended)

### 1. Clone and enter the repo

```bash
git clone <repo-url>
cd church
```

### 2. Generate JWT keys (one-time)

```bash
mkdir -p backend/keys
openssl genrsa -out backend/keys/private.pem 2048
openssl rsa -in backend/keys/private.pem -pubout -out backend/keys/public.pem
```

### 3. Configure the backend environment

```bash
cp backend/.env.example backend/.env
```

The defaults in `.env.example` work out of the box for Docker Compose (database URL points to the `db` service). Fill in the optional services only if you need them:

```env
# Required for emails
RESEND_API_KEY=re_...
EMAIL_FROM=noreply@yourdomain.com

# Required for file uploads
R2_ACCOUNT_ID=...
R2_ACCESS_KEY_ID=...
R2_SECRET_ACCESS_KEY=...
R2_BUCKET_NAME=igreja-organizada
R2_PUBLIC_URL=https://pub-xxx.r2.dev
```

### 4. Start all services

```bash
docker compose -f docker-compose.yml up --build
```

This will:
1. Start PostgreSQL 16
2. Run database migrations automatically
3. Start the Go API on port 8080
4. Build and start the Next.js frontend on port 3000

Open http://localhost:3000 — the app is ready.

---

## Development with Hot Reload

Use the dev compose file, which extends the production one with bind-mounted source directories and hot-reload tooling (`air` for Go, `next dev` for frontend).

```bash
# Start the database first (from docker-compose.yml)
docker compose -f docker-compose.yml up db -d

# Start backend and frontend with hot reload
docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build
```

- Backend: [air](https://github.com/air-verse/air) watches `backend/` and rebuilds on change
- Frontend: `next dev` with HMR on port 3000
- Go module cache is preserved in a named volume (`go_module_cache`) to speed up rebuilds
- `node_modules` inside the frontend container is kept isolated from the host bind mount

---

## Running Locally Without Docker

### Backend

Requires Go 1.25+ and a running PostgreSQL instance.

```bash
cd backend

# Install golang-migrate CLI (once)
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Copy and edit env
cp .env.example .env
# Edit DATABASE_URL to point to your local Postgres:
# DATABASE_URL=postgres://church:church@localhost:5432/igreja_organizada?sslmode=disable

# Run migrations
migrate -source file://db/migrations -database "$DATABASE_URL" up

# Run the server
go run ./cmd/api
```

The API will be available at http://localhost:8080.

### Frontend

Requires Node.js 20+.

```bash
cd frontend

# Install dependencies
npm install

# Copy env
cp .env.example .env.local
# NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1

# Start dev server
npm run dev
```

The app will be available at http://localhost:3000.

---

## Environment Variables

### Backend — `backend/.env`

| Variable                | Default (Docker)                                                    | Description                              |
|-------------------------|---------------------------------------------------------------------|------------------------------------------|
| `DATABASE_URL`          | `postgres://church:church@db:5432/igreja_organizada?sslmode=disable` | PostgreSQL connection string             |
| `JWT_PRIVATE_KEY_PATH`  | `/app/keys/private.pem`                                             | Path to RSA private key (PEM)            |
| `JWT_PUBLIC_KEY_PATH`   | `/app/keys/public.pem`                                              | Path to RSA public key (PEM)             |
| `JWT_ACCESS_TTL_MINUTES`| `15`                                                                | Access token TTL in minutes              |
| `JWT_REFRESH_TTL_DAYS`  | `30`                                                                | Refresh token TTL in days                |
| `RESEND_API_KEY`        | —                                                                   | Resend.com API key (emails optional)     |
| `EMAIL_FROM`            | —                                                                   | Sender address for transactional emails  |
| `R2_ACCOUNT_ID`         | —                                                                   | Cloudflare R2 account ID (uploads optional) |
| `R2_ACCESS_KEY_ID`      | —                                                                   | R2 access key                            |
| `R2_SECRET_ACCESS_KEY`  | —                                                                   | R2 secret key                            |
| `R2_BUCKET_NAME`        | `igreja-organizada`                                                 | R2 bucket name                           |
| `R2_PUBLIC_URL`         | —                                                                   | Public base URL for R2 assets            |
| `PORT`                  | `8080`                                                              | HTTP server port                         |
| `ENV`                   | `development`                                                       | `development` or `production`            |
| `LOG_LEVEL`             | `info`                                                              | `debug`, `info`, `warn`, or `error`      |

### Frontend — `frontend/.env.local`

| Variable               | Default                           | Description               |
|------------------------|-----------------------------------|---------------------------|
| `NEXT_PUBLIC_API_URL`  | `http://localhost:8080/api/v1`    | Backend API base URL      |

---

## JWT Keys

The backend requires an RSA-2048 key pair for JWT RS256 signing. Keys live in `backend/keys/` and are mounted read-only into the container.

```bash
# Generate (one-time, from repo root)
mkdir -p backend/keys
openssl genrsa -out backend/keys/private.pem 2048
openssl rsa -in backend/keys/private.pem -pubout -out backend/keys/public.pem
```

`backend/keys/` is in `.gitignore` — never commit the private key.

---

## Database Migrations

Migration files live in `backend/db/migrations/` and follow the `golang-migrate` naming convention: `NNNN_description.up.sql` / `NNNN_description.down.sql`.

Migrations run automatically on container startup (via `docker-entrypoint.sh`). To run them manually:

```bash
# Up
migrate -source file://backend/db/migrations -database "$DATABASE_URL" up

# Down (one step)
migrate -source file://backend/db/migrations -database "$DATABASE_URL" down 1
```

### Current schema (migration 0001)

Core tables: `churches`, `members`, `member_church_memberships`, `roles`, `member_roles`, `instruments`, `member_instruments`, `refresh_tokens`.

All primary keys are UUID v4 (`gen_random_uuid()`). Multi-tenant isolation is enforced via `church_id` on every domain table.

---

## API Reference

Base path: `/api/v1`  
Full contract: [`contracts/openapi.yaml`](contracts/openapi.yaml)

### Authentication

| Method | Path                    | Auth     | Description                            |
|--------|-------------------------|----------|----------------------------------------|
| POST   | `/auth/register`        | public   | Register a new church + pastor account |
| POST   | `/auth/login`           | public   | Login → access token + refresh cookie  |
| POST   | `/auth/refresh`         | cookie   | Rotate refresh token, new access token |
| POST   | `/auth/logout`          | any      | Revoke current refresh token           |
| POST   | `/auth/logout-all`      | any      | Revoke all sessions for this member    |

### Members

| Method | Path                              | Access      | Description                       |
|--------|-----------------------------------|-------------|-----------------------------------|
| GET    | `/members`                        | Leadership+ | List members (search + pagination)|
| POST   | `/members`                        | Leadership+ | Create member                     |
| POST   | `/members/import`                 | Leadership+ | Bulk CSV import                   |
| GET    | `/members/me`                     | Any         | Own profile                       |
| PUT    | `/members/me`                     | Any         | Update own profile                |
| GET    | `/members/me/instruments`         | Any         | Own instruments                   |
| POST   | `/members/me/instruments`         | Any         | Add instrument                    |
| DELETE | `/members/me/instruments/{id}`    | Any         | Remove instrument                 |
| GET    | `/members/{id}`                   | Leadership+ | Member detail                     |
| PUT    | `/members/{id}`                   | Leadership+ | Update member                     |
| DELETE | `/members/{id}`                   | Pastor      | Deactivate member                 |
| GET    | `/members/{id}/roles`             | Leadership+ | Roles assigned to member          |
| POST   | `/members/{id}/roles`             | Leadership+ | Assign role                       |
| DELETE | `/members/{id}/roles/{role_id}`   | Leadership+ | Remove role                       |

### Roles & Instruments

| Method | Path                 | Access      | Description                       |
|--------|----------------------|-------------|-----------------------------------|
| GET    | `/roles`             | Leadership+ | List all roles                    |
| POST   | `/roles`             | Pastor      | Create custom role                |
| PUT    | `/roles/{id}`        | Pastor      | Update custom role                |
| DELETE | `/roles/{id}`        | Pastor      | Delete custom role                |
| GET    | `/instruments`       | Any         | List all instruments              |
| POST   | `/instruments`       | Leadership+ | Create custom instrument          |
| DELETE | `/instruments/{id}`  | Leadership+ | Delete custom instrument          |

### Schedules (Worship)

| Method | Path                                  | Access      | Description                       |
|--------|---------------------------------------|-------------|-----------------------------------|
| GET    | `/schedules`                          | Any         | List schedules (paginated)        |
| POST   | `/schedules`                          | Leadership+ | Create schedule (draft)           |
| GET    | `/schedules/{id}`                     | Any         | Schedule detail with slots        |
| PUT    | `/schedules/{id}`                     | Leadership+ | Update schedule metadata          |
| DELETE | `/schedules/{id}`                     | Leadership+ | Cancel schedule                   |
| POST   | `/schedules/{id}/publish`             | Leadership+ | Publish → email all slots         |
| GET    | `/schedules/{id}/slots`               | Any         | List schedule slots               |
| POST   | `/schedules/{id}/slots`               | Leadership+ | Add member to schedule            |
| DELETE | `/schedules/{id}/slots/{slot_id}`     | Leadership+ | Remove member from schedule       |
| POST   | `/schedules/{id}/slots/{slot_id}/confirm` | Musician+ | Member confirms own slot      |
| GET    | `/schedules/suggest/{sunday_date}`    | Leadership+ | Auto-suggest lineup for a Sunday  |

### Availability

| Method | Path                               | Access      | Description                       |
|--------|------------------------------------|-------------|-----------------------------------|
| GET    | `/availability/exceptions`         | Any         | Own unavailability (current month)|
| POST   | `/availability/exceptions`         | Any         | Mark Sunday as unavailable        |
| DELETE | `/availability/exceptions/{id}`    | Any         | Remove unavailability             |
| GET    | `/availability/exceptions/all`     | Leadership+ | All church exceptions             |

### Pastoral Agenda

| Method | Path                              | Access      | Description                       |
|--------|-----------------------------------|-------------|-----------------------------------|
| GET    | `/agenda/slots`                   | Any         | Pastor availability slots         |
| POST   | `/agenda/slots`                   | Pastor      | Create availability slot          |
| PUT    | `/agenda/slots/{id}`              | Pastor      | Update slot                       |
| DELETE | `/agenda/slots/{id}`              | Pastor      | Remove slot                       |
| GET    | `/agenda/events`                  | Any         | List appointments                 |
| POST   | `/agenda/events`                  | Any         | Request appointment               |
| GET    | `/agenda/events/{id}`             | Any         | Appointment detail                |
| PUT    | `/agenda/events/{id}`             | Pastor      | Edit appointment                  |
| POST   | `/agenda/events/{id}/confirm`     | Pastor      | Confirm → send calendar invite    |
| POST   | `/agenda/events/{id}/decline`     | Pastor      | Decline → email requester         |
| POST   | `/agenda/events/{id}/cancel`      | Pastor      | Cancel confirmed appointment      |

### Inventory

| Method | Path                              | Access      | Description                       |
|--------|-----------------------------------|-------------|-----------------------------------|
| GET    | `/inventory/categories`           | Any         | List categories                   |
| POST   | `/inventory/categories`           | Leadership+ | Create category                   |
| GET    | `/inventory/items`                | Any         | List items                        |
| POST   | `/inventory/items`                | Leadership+ | Create item                       |
| GET    | `/inventory/items/{id}`           | Any         | Item detail                       |
| PUT    | `/inventory/items/{id}`           | Leadership+ | Update item                       |
| POST   | `/inventory/items/{id}/photo`     | Leadership+ | Upload photo (multipart)          |
| POST   | `/inventory/items/{id}/discard`   | Leadership+ | Mark as discarded                 |
| POST   | `/inventory/items/{id}/donate`    | Leadership+ | Mark as donated                   |
| GET    | `/inventory/loans`                | Leadership+ | List loans                        |
| POST   | `/inventory/loans`                | Any         | Request a loan                    |
| POST   | `/inventory/loans/{id}/approve`   | Leadership+ | Approve loan                      |
| POST   | `/inventory/loans/{id}/reject`    | Leadership+ | Reject loan                       |
| POST   | `/inventory/loans/{id}/return`    | Leadership+ | Register return                   |

### Response shapes

**Error envelope:**
```json
{
  "error": {
    "code": "MEMBER_NOT_FOUND",
    "message": "Membro não encontrado.",
    "field": null
  }
}
```

**List envelope:**
```json
{
  "data": [...],
  "meta": { "total": 48, "page": 1, "per_page": 20 }
}
```

**Permission levels:** `P` = Pastor · `L` = Leadership+ · `M` = Musician+ · `*` = any authenticated

---

## Project Structure

```
backend/
├── cmd/api/main.go                  # Entry point — wires dependencies
├── internal/
│   ├── config/config.go             # envconfig struct
│   ├── domain/                      # Pure business entities (no I/O)
│   ├── ports/
│   │   ├── repository.go            # DB interface definitions + sentinel errors
│   │   ├── mailer.go                # Email port interface
│   │   └── storage.go               # File storage port interface
│   ├── adapters/
│   │   ├── postgres/                # pgx repository implementations
│   │   │   ├── queries/             # .sql files (sqlc input)
│   │   │   ├── generated/           # sqlc output — DO NOT EDIT
│   │   │   ├── auth_repo.go
│   │   │   └── member_repo.go
│   │   ├── resend/                  # Email adapter (stub)
│   │   └── r2/                      # Storage adapter (stub)
│   ├── handlers/                    # HTTP handlers (parse → service → respond)
│   │   ├── auth.go
│   │   ├── members.go
│   │   ├── roles.go
│   │   ├── instruments.go
│   │   ├── middleware.go            # JWT auth, RequireProfile
│   │   └── response.go             # writeJSON, writeError helpers
│   └── services/                   # Business logic + orchestration
│       ├── auth_service.go
│       └── member_service.go
├── db/migrations/                   # golang-migrate SQL files
├── keys/                            # RSA key pair (gitignored)
├── Dockerfile                       # Multi-stage production build
├── Dockerfile.dev                   # Hot-reload dev build (air)
├── docker-entrypoint.sh             # Runs migrations then starts server
└── .env.example

frontend/
├── app/
│   ├── (auth)/
│   │   ├── login/page.tsx           # Login form
│   │   └── register/page.tsx        # Church registration form
│   ├── (dashboard)/
│   │   ├── layout.tsx               # Dashboard shell + nav
│   │   └── members/
│   │       ├── page.tsx             # Member list
│   │       ├── new/page.tsx         # Create member form
│   │       └── [id]/page.tsx        # Member detail
│   ├── layout.tsx                   # Root layout (fonts, providers)
│   └── page.tsx                     # Redirect → /members
├── components/
│   ├── ui/                          # shadcn/ui primitives (do not modify)
│   └── features/                    # Feature-specific components
│       ├── DashboardNav.tsx
│       └── members/
├── hooks/
│   ├── useMembers.ts                # TanStack Query hooks for members domain
│   └── useDebounce.ts
├── lib/
│   ├── api.ts                       # Typed fetch wrapper (auth headers, 401 retry)
│   ├── auth.ts                      # Access token in memory, session helpers
│   └── utils.ts                     # cn() helper
├── Dockerfile                       # Production build
├── Dockerfile.dev                   # next dev
└── .env.example

contracts/
├── openapi.yaml                     # Single source of truth for all API contracts
├── ARCHITECTURE.md                  # Stack decisions, algorithms, conventions
└── UI_UX_STANDARDS.md               # Frontend quality bar
```

---

## Implemented so far

### Backend
- Auth: register, login, refresh, logout, logout-all (JWT RS256 + HttpOnly cookie)
- Members: full CRUD + bulk import + `/me` profile
- Roles: list, create, update, delete (system roles are immutable)
- Instruments: list, create, delete
- Member roles: assign, remove
- Member instruments: list, add, remove

### Frontend
- Login and church registration pages
- Members list (search, filter by role, pagination)
- Member detail (roles, instruments, deactivate)
- Create member form
- Dashboard shell with nav (mobile bottom bar + desktop sidebar)
