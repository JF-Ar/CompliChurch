# Igreja Organizada — Frontend

## mandatory first steps
Before writing any code, read these two files completely:
- `/contracts/ARCHITECTURE.md` — stack decisions, folder structure, auth flow, API conventions
- `/contracts/openapi.yaml` — all endpoint contracts, request/response types

Do not invent endpoints, field names or types not documented in those files.

## project
Church management SaaS MVP. Next.js frontend, mobile-first PWA.

## stack
- Next.js 14+ App Router
- TypeScript (strict mode)
- Tailwind CSS — no inline styles, no CSS modules
- shadcn/ui for primitive components
- TanStack Query (React Query) for all data fetching and mutations
- React Hook Form + Zod for forms and validation
- API client lives in `lib/api.ts` — never call `fetch()` directly in components

## api
- Base URL from `NEXT_PUBLIC_API_URL` env var
- All API calls go through `lib/api.ts`
- Types are derived from `contracts/openapi.yaml`
- Do not call endpoints not listed in ARCHITECTURE.md section 5

## auth
- Access token stored in memory (JS variable) — never localStorage, never sessionStorage
- Refresh token is an HttpOnly cookie handled automatically by the browser
- On 401 response, call `POST /auth/refresh` once, then retry the original request
- If refresh also fails, redirect to `/login`
- Auth helpers live in `lib/auth.ts`

## folder structure
```
app/
  (auth)/
    login/
    register/
  (dashboard)/
    agenda/
    schedule/
    inventory/
    members/
  layout.tsx
  page.tsx
components/
  ui/        shadcn/ui primitives
  features/  feature-specific components
lib/
  api.ts     typed API client
  auth.ts    auth helpers
```

## environment variables
```
NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1
```

## conventions
- All pages are server components by default; add `"use client"` only when needed
- TanStack Query hooks live in `hooks/` or colocated with the feature component
- Zod schemas mirror the request bodies from `contracts/openapi.yaml`
- All dates displayed in pt-BR locale (`dd/MM/yyyy`)
- UUIDs are never shown to the user

## do not
- Call `fetch()` directly in components — use `lib/api.ts`
- Store tokens in localStorage or sessionStorage
- Call endpoints not listed in ARCHITECTURE.md section 5
- Use inline styles — Tailwind classes only
- Create components that duplicate shadcn/ui primitives