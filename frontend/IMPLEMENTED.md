# Frontend — Implemented Pages, Hooks & API Functions

Quick reference for agents. Read this only to locate a specific page or check whether a feature exists.

---

## Auth Token Strategy

- Access token: module-level `let accessToken` in lib/auth.ts — memory only, never persisted
- Refresh token: HttpOnly cookie — browser sends automatically, never read/written in JS
- 401 handling: apiFetch() calls doRefresh() once, retries; on second 401 throws UNAUTHORIZED
  and clearSession() wipes in-memory state
- Session state: `getAccessToken()`, `setAccessToken()`, `setSession(token, member, church)`,
  `getSession()`, `clearSession()` — all in lib/auth.ts

---

## Auth Pages

- Login → app/(auth)/login/page.tsx
  - API: login() — raw fetch (no auth header needed)
  - On success: setSession(token, member, church) → router.push("/members")
  - Link to /register at bottom

- Register → app/(auth)/register/page.tsx
  - API: register() — raw fetch, POST /auth/register
  - Creates church + pastor in one step
  - 409 → field error on email; on success stores session + redirects to /members

---

## Members

- Member list → app/(dashboard)/members/page.tsx
  - Hook: useMembers(params) from hooks/useMembers.ts
  - Features: debounced name search, base_profile filter, is_active toggle, pagination
  - Components: MemberCard, MemberCardSkeleton, RoleBadge
  - Leadership+ sees deactivate action

- Member detail → app/(dashboard)/members/[id]/page.tsx
  - Hooks: useMember, useRoles, useAssignRole, useRemoveRole,
    useMemberInstruments, useAddMemberInstrument, useRemoveMemberInstrument
  - Inline edit form (name/phone/birth_date) via useUpdateMember
  - Leadership+: assign/remove roles, add/remove instruments
  - Deactivate: confirm dialog → useDeactivateMember

- New member → app/(dashboard)/members/new/page.tsx
  - Hook: useCreateMember, useRoles
  - Fields: name, email, phone, birth_date
  - Role selector: collapsible checkbox list with RoleBadge; role_ids sent in body
  - On success: redirect to /members/{id}

- Own profile → app/(dashboard)/members/me/page.tsx
  - Hooks: useMe, useUpdateMe, useMyInstruments, useAddInstrument, useRemoveInstrument,
    useInstruments
  - Inline edit for name/phone/birth_date
  - Instruments: add from catalog, remove; first added → is_primary: true

---

## Inventory

- Item list → app/(dashboard)/inventory/page.tsx
  - Hook: useItems(params), useCategories, useMe
  - Filters: search (debounced), category, status, item_type
  - include_deleted toggle (leadership+ only)
  - Status badges: available=green, on_loan=amber, maintenance=red
  - Donation/Discarded badge shown when item has deletion_reason
  - Pagination; buttons: "+ Novo item" + "Empréstimos" (leadership+)

- New item → app/(dashboard)/inventory/new/page.tsx
  - Hook: useCreateItem, useCategories, useCreateCategory
  - Fields: name, item_type (radio), category dropdown, description, asset_number,
    location, quantity, qty_min_alert (consumable only), serial_number, notes
  - Inline category creation: "+ Nova categoria" sentinel opens Dialog mini-form;
    on success invalidates categories and auto-selects new category
  - On success: redirect to /inventory/{id}

- Item detail → app/(dashboard)/inventory/[id]/page.tsx
  - Hooks: useItem, useUpdateItem, useUploadItemPhoto, useDiscardItem, useDonateItem,
    useLoans, useCreateLoan, useCongregations, useMembers, useMe
  - Sections: photo upload (leadership+), all fields, inline edit form (leadership+)
  - Discard + donate: confirm dialogs (leadership+), shows Doado/Descartado badge
  - Loans section: filtered client-side from full loan list
  - New loan modal: member/congregation selector

- Loans list → app/(dashboard)/inventory/loans/page.tsx
  - Hooks: useLoans(params), useApproveLoan, useRejectLoan, useReturnLoan, useMe
  - Leadership gate for action buttons (LoanActions + LoanCard)
  - Status filter; mobile cards + desktop table layout
  - Approve/reject shown for pending; return modal (return_condition enum) for active

---

## Navigation & Layout

- Dashboard shell → app/(dashboard)/layout.tsx
- Navigation → components/features/DashboardNav.tsx
  - Bottom nav (mobile) + sidebar (desktop)
  - Routes: /members, /schedule (placeholder), /agenda (placeholder), /inventory, /members/me
  - Active state: pathname.startsWith(href) except /members/me excluded from /members

---

## API Client — lib/api.ts

All calls go through `apiFetch<T>(path, options)` except login() and register() (raw fetch).

### Auth
- `login(email, password)` → POST /auth/login
- `register(body)` → POST /auth/register
- `logout()` → POST /auth/logout
- `logoutAll()` → POST /auth/logout-all

### Churches
- `getMyChurch()` → GET /churches/me
- `listCongregations()` → GET /churches/me/congregations

### Members
- `getMe()` → GET /members/me
- `updateMe(data)` → PUT /members/me
- `listMembers(params?)` → GET /members
- `getMember(id)` → GET /members/{id}
- `createMember(data)` → POST /members
- `updateMember(id, data)` → PUT /members/{id}
- `deactivateMember(id)` → DELETE /members/{id}
- `getMyInstruments()` → GET /members/me/instruments
- `addMyInstrument(data)` → POST /members/me/instruments
- `removeMyInstrument(instrumentId)` → DELETE /members/me/instruments/{id}
- `getMemberInstruments(memberId)` → GET /members/{id}/instruments
- `addMemberInstrument(memberId, data)` → POST /members/{id}/instruments
- `removeMemberInstrument(memberId, instrumentId)` → DELETE /members/{id}/instruments/{id}
- `assignRole(memberId, roleId)` → POST /members/{id}/roles
- `removeRole(memberId, roleId)` → DELETE /members/{id}/roles/{roleId}

### Roles & Instruments
- `listRoles()` → GET /roles
- `createRole(data)` → POST /roles
- `updateRole(id, data)` → PUT /roles/{id}
- `deleteRole(id)` → DELETE /roles/{id}
- `listInstruments()` → GET /instruments
- `createInstrument(data)` → POST /instruments
- `deleteInstrument(id)` → DELETE /instruments/{id}

### Schedules _(functions defined, no UI pages yet)_
- `listSchedules(params?)` → GET /schedules
- `getSchedule(id)` → GET /schedules/{id}
- `createSchedule(data)` → POST /schedules
- `getScheduleSuggestion(sundayDate)` → GET /schedules/suggest/{date}
- `publishSchedule(id)` → POST /schedules/{id}/publish
- `addScheduleSlot(scheduleId, data)` → POST /schedules/{id}/slots
- `removeScheduleSlot(scheduleId, slotId)` → DELETE /schedules/{id}/slots/{slotId}
- `confirmScheduleSlot(scheduleId, slotId)` → POST /schedules/{id}/slots/{slotId}/confirm

### Availability _(functions defined, no UI pages yet)_
- `listMyExceptions(month?)` → GET /availability/exceptions
- `createException(data)` → POST /availability/exceptions
- `deleteException(id)` → DELETE /availability/exceptions/{id}

### Agenda _(functions defined, no UI pages yet)_
- `listEvents(params?)` → GET /agenda/events

### Inventory
- `listCategories()` → GET /inventory/categories
- `createCategory(data)` → POST /inventory/categories
- `updateCategory(id, data)` → PUT /inventory/categories/{id}
- `deleteCategory(id)` → DELETE /inventory/categories/{id}
- `listItems(params?)` → GET /inventory/items
- `createItem(data)` → POST /inventory/items
- `getItem(id)` → GET /inventory/items/{id}
- `updateItem(id, data)` → PUT /inventory/items/{id}
- `uploadItemPhoto(id, file)` → POST /inventory/items/{id}/photo
- `discardItem(id)` → POST /inventory/items/{id}/discard
- `donateItem(id)` → POST /inventory/items/{id}/donate
- `listLoans(params?)` → GET /inventory/loans
- `createLoan(data)` → POST /inventory/loans
- `getLoan(id)` → GET /inventory/loans/{id}
- `approveLoan(id)` → POST /inventory/loans/{id}/approve
- `rejectLoan(id)` → POST /inventory/loans/{id}/reject
- `returnLoan(id, data)` → POST /inventory/loans/{id}/return

---

## Hooks — hooks/useMembers.ts

Members: `useMembers`, `useMember`, `useCreateMember`, `useUpdateMember`, `useDeactivateMember`
Me: `useMe`, `useUpdateMe`
Own instruments: `useMyInstruments`, `useAddInstrument`, `useRemoveInstrument`
Member instruments (leadership): `useMemberInstruments`, `useAddMemberInstrument`, `useRemoveMemberInstrument`
Roles: `useRoles`, `useAssignRole(memberId)`, `useRemoveRole(memberId)`
Instruments catalog: `useInstruments`

## Hooks — hooks/useInventory.ts

Categories: `useCategories`, `useCreateCategory`, `useUpdateCategory`, `useDeleteCategory`
Items: `useItems`, `useItem`, `useCreateItem`, `useUpdateItem`, `useUploadItemPhoto`,
       `useDiscardItem`, `useDonateItem`
Loans: `useLoans`, `useLoan`, `useCreateLoan`, `useApproveLoan`, `useRejectLoan`, `useReturnLoan`
Churches: `useCongregations`

## Hooks — hooks/useDebounce.ts

`useDebounce<T>(value, delay)` — generic debounce, used in member and item search inputs

---

## UI Primitives — components/ui/

- button.tsx — cva variants: default, destructive, outline, secondary, ghost, link
- input.tsx — forwarded ref input
- label.tsx — forwarded ref label
- badge.tsx — variants: default, secondary, destructive, outline, success, warning, muted
- skeleton.tsx — animate-pulse div
- dialog.tsx — Radix Dialog (Dialog, DialogContent, DialogHeader, DialogTitle,
  DialogDescription, DialogFooter, DialogClose, DialogTrigger)
- sonner.tsx — Sonner toast wrapper, position bottom-center

## Feature Components — components/features/

- members/MemberCard.tsx — list card with avatar initials, roles, active badge
- members/MemberCardSkeleton.tsx — loading skeleton
- members/RoleBadge.tsx — colored badge per base_profile
- members/DeactivateDialog.tsx — confirm dialog for member deactivation
- DashboardNav.tsx — bottom nav (mobile) + sidebar (desktop)

---

## App Infrastructure

- app/providers.tsx — QueryClientProvider (TanStack Query v5) + Sonner Toaster
- app/layout.tsx — Geist Sans + Geist Mono fonts, h-full antialiased
- app/(dashboard)/layout.tsx — dashboard shell with DashboardNav
- lib/auth.ts — token state: accessToken, currentMember, currentChurch module vars
- lib/utils.ts — cn() (clsx + tailwind-merge)
