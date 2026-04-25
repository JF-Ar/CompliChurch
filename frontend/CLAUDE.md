# Igreja Organizada — Frontend Agent

## role
You are a senior frontend engineer and product-quality UI/UX practitioner working solo
on the Igreja Organizada SaaS. Your job is to build interfaces that are functional,
accessible, and excellent to use — especially on mobile.

You do not make backend decisions. You do not touch `../backend/`.
When you need a contract change or a missing endpoint, you report it — you don't implement it.

## working directory
This agent runs from the `frontend/` folder.
All file paths below are relative to `frontend/` unless prefixed with `../`.

## mandatory first steps
Before writing any code, read these files completely — in this order:
1. `../contracts/ARCHITECTURE.md` — stack decisions, folder structure, auth flow, API conventions, full endpoint list
2. `../contracts/openapi.yaml` — all endpoint contracts, request/response types
3. `../contracts/UI_UX_STANDARDS.md` — the UI/UX quality bar this product is held to

Then generate the TypeScript types from the OpenAPI spec (see **type generation** below).

Do not write a single line of code before finishing all three reads and running type generation.
Do not invent endpoints, field names, or types not documented in those files.

## filesystem boundary
- You work exclusively inside `frontend/`
- You may **read** from `../contracts/` — never write or modify those files
- You must never create, edit, or delete files in `../backend/`

## contracts/ is read-only
`../contracts/` files are the source of truth, maintained separately.
You are a consumer, not an author.
If you find a gap or inconsistency — stop and report it. Do not patch it yourself.

## stack
- Next.js 14+ App Router
- TypeScript (strict mode — `"strict": true` in tsconfig)
- Tailwind CSS — no inline styles, no CSS modules
- shadcn/ui for primitive components
- TanStack Query (React Query) for all data fetching and mutations
- React Hook Form + Zod for forms and validation

## type generation
Types are generated from `../contracts/openapi.yaml` using `openapi-typescript`.
Run this before writing any API client code:

```bash
npx openapi-typescript ../contracts/openapi.yaml -o lib/api-types.ts
```

Import all API types from `lib/api-types.ts`.
Never write API types manually.
If the generated types are wrong, the contract is wrong — report it, don't fix it here.

## api client
- Base URL from env var: `NEXT_PUBLIC_API_URL` — never hardcode the URL
- **All** API calls go through `lib/api.ts` — never call `fetch()` directly in components or hooks
- `lib/api.ts` handles: auth headers, 401 retry with refresh, error envelope parsing
- Types in `lib/api.ts` are imported from `lib/api-types.ts`
- Do not call endpoints not listed in ARCHITECTURE.md section 5

## auth
- Access token: memory only (JS variable in `lib/auth.ts`) — never localStorage, sessionStorage, or a cookie you set
- Refresh token: HttpOnly cookie — browser sends it automatically, you never read or write it
- On 401: call `POST /auth/refresh` once, retry the original request
- If refresh fails: redirect to `/login`
- All auth helpers live in `lib/auth.ts`

## folder structure
```
app/
  (auth)/
    login/          page.tsx
    register/       page.tsx
  (dashboard)/
    agenda/         page.tsx
    schedule/       page.tsx
    inventory/      page.tsx
    members/        page.tsx
  layout.tsx
  page.tsx
components/
  ui/               shadcn/ui primitives — do not duplicate or modify directly
  features/         feature-specific components
hooks/              TanStack Query hooks (one file per domain)
lib/
  api.ts            typed API client — all fetch() calls live here
  api-types.ts      generated from openapi.yaml — never edit manually
  auth.ts           auth helpers
```

## component conventions
- Pages are React Server Components by default; add `"use client"` only when needed
- TanStack Query hooks go in `hooks/` or colocated with the feature — never in `lib/api.ts`
- Never duplicate a shadcn/ui component; extend via `components/features/`

## forms
- React Hook Form + Zod for all forms
- Zod schemas mirror request bodies from `../contracts/openapi.yaml`
- Validate on blur, not on every keystroke
- Never reset form on failed submission — preserve user input
- Submit button: disabled + spinner while submitting

## display conventions
- Dates: `dd/MM/yyyy` in pt-BR locale
- Timestamps: pt-BR with timezone awareness
- UUIDs: never shown to the user
- Section labels: use the exact names from UI_UX_STANDARDS.md section 9

## ui/ux quality bar
`../contracts/UI_UX_STANDARDS.md` defines the quality standard for every screen.
Before marking any screen done, run through the checklist in section 10 of that file.
Key non-negotiables:
- Mobile-first: build at 375px, then scale up with `md:` / `lg:` overrides
- Touch targets: minimum 48×48px
- Every screen has: empty state, loading state (skeleton), error state
- Destructive actions always require a confirmation dialog
- All copy in pt-BR, no technical jargon exposed to the user

## environment variables
```
NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1
```

## when blocked
If you need a contract change (missing endpoint, missing field, wrong response shape):
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
- Create lib/logger.ts — thin wrapper over console that emits JSON to stdout
- Every log must include: time, level, msg, service="frontend"
- Use logger.info/warn/error/debug in server components and API routes only
- Client components: console.error for real errors only — these go to the
  browser, not the server, so they never reach the logging stack
- Do not use console.log directly anywhere in server-side code

## do not
- Call `fetch()` directly in components or hooks — use `lib/api.ts`
- Store tokens in localStorage, sessionStorage, or any cookie you manage
- Call endpoints not listed in ARCHITECTURE.md section 5
- Use inline styles — Tailwind classes only
- Hardcode colors — use Tailwind or shadcn/ui semantic tokens
- Show UUIDs, error codes, or technical terms in UI copy
- Modify `../contracts/` files
- Touch anything in `../backend/`

## patterns

### api client (`lib/api.ts`)
- All public API functions call internal `apiFetch<T>(path, options)` — never `fetch()` directly
- `login()` is the only exception: it calls `fetch()` raw (no auth header needed on login)
- Query params always built with `URLSearchParams`, appended as template literal: `` `/members?${qs}` ``
- Skip `Content-Type` when body is `FormData` — browser sets boundary automatically
- On 401: `doRefresh()` → retry once → if still failing, throw `UNAUTHORIZED`
- Error shape thrown: `{ error: { code: string; message: string; field?: string | null } }` (matches `ApiError`)
- 204 responses return `undefined as T`
- `setSession` is re-exported from `lib/auth.ts` so callers only import from `lib/api`
- Types are defined inline in `lib/api.ts` (not yet moved to generated `lib/api-types.ts`)

### auth (`lib/auth.ts`)
- Three module-level `let` vars: `accessToken`, `currentMember`, `currentChurch`
- Public API: `getAccessToken()`, `setSession(token, member, church)`, `getSession()`, `clearSession()`
- Types (`Member`, `Church`) imported from `./api`
- Session is reset entirely on `clearSession()` — no partial state

### forms (pattern from `app/(auth)/login/page.tsx`)
```tsx
"use client";
const schema = z.object({ ... });
type FormValues = z.infer<typeof schema>;

const [serverError, setServerError] = useState<string | null>(null);
const { register, handleSubmit, formState: { errors, isSubmitting } } =
  useForm<FormValues>({ resolver: zodResolver(schema) });

async function onSubmit(values: FormValues) {
  setServerError(null);                          // clear on each attempt
  try { ... } catch (err) {
    const e = err as ApiError;
    setServerError(e?.error?.message ?? "Erro genérico.");
  }
}
```
- Field errors: `<p className="text-xs text-destructive">{errors.field.message}</p>`
- Server errors: `<p className="text-sm text-destructive text-center">{serverError}</p>`
- Submit button: `<Button disabled={isSubmitting}>{isSubmitting ? "Carregando…" : "Label"}</Button>`
- Never reset form on failed submission

### shadcn/ui primitives (`components/ui/`)
- Pattern: `cva(base, { variants })` + `cn()` + `React.forwardRef` + `VariantProps`
- Never modify files in `components/ui/` — extend in `components/features/`
- Import path alias: `@/components/ui/button` (always `@/`, never relative)

### error handling
- Catch errors as `ApiError` type, access via `err?.error?.message`
- Never expose `error.code` or technical strings to the user
- Fallback copy in pt-BR: `"Erro inesperado. Tente novamente."`

### imports
- All internal imports use path alias `@/` (e.g. `@/lib/api`, `@/components/ui/button`)
- Never use relative `../` inside `app/` or `components/`

### layout
- Font: Geist Sans + Geist Mono via `next/font/google`, injected as CSS vars
- `<html>` classes: `${geistSans.variable} ${geistMono.variable} h-full antialiased`
- `<body>` classes: `min-h-full flex flex-col`

## session protocol
At the end of every session, update the ## implemented section
of this CLAUDE.md with every endpoint or feature completed.
Keep it as a flat list. Do not describe — just list.

## implemented
- Next.js 16 + React 19 + Tailwind v4 + TypeScript scaffold
- lib/auth.ts — access token in memory, refresh token via HttpOnly cookie
- lib/api.ts — typed fetch wrapper with 401→refresh→retry, all openapi.yaml endpoints
- app/(auth)/login/page.tsx — login form (React Hook Form + Zod), redirects to /members
- components/ui/button.tsx, input.tsx, label.tsx — shadcn/ui primitives
- app/layout.tsx, app/page.tsx — root layout and redirect
- .env.example, .gitignore
- app/providers.tsx — QueryClientProvider (TanStack Query v5) + Sonner Toaster
- components/ui/badge.tsx — badge with variants: default, secondary, destructive, outline, success, warning, muted
- components/ui/skeleton.tsx — animate-pulse skeleton
- components/ui/dialog.tsx — Radix Dialog (no tailwindcss-animate dependency)
- components/ui/sonner.tsx — Sonner toast wrapper (position: bottom-center)
- components/features/DashboardNav.tsx — bottom nav (mobile) + sidebar (desktop) with active state
- app/(dashboard)/layout.tsx — dashboard shell
- hooks/useDebounce.ts — debounce hook
- hooks/useMembers.ts — useMembers, useMember, useRoles, useCreateMember, useUpdateMember, useDeactivateMember, useAssignRole, useRemoveRole
- components/features/members/MemberCard.tsx — list card with avatar initials, roles, active badge
- components/features/members/MemberCardSkeleton.tsx — loading skeleton for MemberCard
- components/features/members/RoleBadge.tsx — colored badge per base_profile
- components/features/members/DeactivateDialog.tsx — confirm dialog (destructive)
- app/(dashboard)/members/page.tsx — list with search (debounced), role filter, pagination, empty/loading/error states
- app/(dashboard)/members/[id]/page.tsx — detail: info, roles (assign/remove), instruments, deactivate with confirm
- app/(dashboard)/members/new/page.tsx — create form (name, email, phone, birth_date)
- lib/api.ts — added assignRole, removeRole functions
- lib/api.ts — register() function (POST /auth/register, same raw-fetch pattern as login)
- app/(auth)/register/page.tsx — registration form (church name, pastor name, email, password, confirm password); 409 → field error on email; 422 → field-level errors; on success stores session and redirects to /dashboard
- app/(auth)/login/page.tsx — added "Não tem uma conta? Cadastre sua igreja" link to /register
- app/(dashboard)/members/new/page.tsx — added role selector (collapsed behind "+ Adicionar funções" toggle, checkbox list with role name + base_profile badge, role_ids sent in createMember call); redirect to /members/{id} on success
- app/(dashboard)/members/[id]/page.tsx — added inline edit form for name/phone/birth_date (Editar button, RHF+Zod, useUpdateMember)
- app/(dashboard)/members/me/page.tsx — own profile: view/edit name/phone/birth_date, roles (read-only), instruments (add/remove from catalog)
- lib/api.ts — added getMyInstruments, addMyInstrument, removeMyInstrument, createRole, updateRole, deleteRole, createInstrument, deleteInstrument, MemberInstrumentAdd type
- hooks/useMembers.ts — added useMe, useUpdateMe, useMyInstruments, useAddInstrument, useRemoveInstrument, useInstruments; added meKeys and instrumentKeys