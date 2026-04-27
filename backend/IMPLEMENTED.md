# Backend — Implemented Endpoints & Rules

Quick reference for agents. Read this only to locate a specific handler or check whether a feature exists.
Every entry follows: `METHOD /path → handlers/file.go:Function`

---

## Auth

- POST /auth/register → handlers/auth.go:Register _(public — not in openapi.yaml)_
  - Creates church + pastor member in a single transaction via AuthRepo.CreateChurchWithPastor
  - bcrypt cost 12; 409 CONFLICT if email taken
  - Returns same shape as login (access_token + member + church)
  - Sets refresh_token HttpOnly cookie

- POST /auth/login → handlers/auth.go:Login _(public)_
  - bcrypt.CompareHashAndPassword → services/auth_service.go:Login
  - Mints RS256 access token (15 min) + refresh token (30 d)
  - Stores refresh JTI in refresh_tokens table
  - Sets refresh_token HttpOnly cookie (Secure + SameSiteStrict)

- POST /auth/refresh → handlers/auth.go:Refresh _(cookie auth)_
  - Reads refresh_token cookie; validates signature + JTI in DB
  - Checks revoked_at IS NULL and expires_at > now
  - Rotates: revokes old JTI, stores new JTI → services/auth_service.go:Refresh

- POST /auth/logout → handlers/auth.go:Logout
  - Revokes current refresh token JTI; clears cookie

- POST /auth/logout-all → handlers/auth.go:LogoutAll
  - Revokes all refresh tokens for auth.MemberID; clears cookie

- Middleware → handlers/middleware.go:Authenticate
  - Extracts Bearer token, validates RS256 signature via AuthService.ValidateAccessToken
  - Injects AuthContext{MemberID, ChurchID, BaseProfile, ChurchIDs} into context

- Per-route auth → handlers/middleware.go:RequireProfile(minProfile)
  - Hierarchy: pastor(4) > leadership(3) > musician(2) > member(1)
  - RequireProfile("leadership") admits pastor and leadership

---

## Members

- GET /members → handlers/members.go:ListMembers _(leadership+)_
  - Filters: search (ILIKE name/email), role (base_profile), is_active, page, per_page
  - Isolation: auth.ChurchID via member_church_memberships JOIN

- POST /members → handlers/members.go:CreateMember _(leadership+)_
  - Validates role_ids belong to church or are system roles
  - 409 MEMBER_EMAIL_EXISTS if email taken

- POST /members/import → handlers/members.go:ImportMembers _(leadership+)_
  - Accepts {members: [{name, email, phone}]}
  - Returns {created, skipped, errors} — never 4xx for row-level failures

- GET /members/me → handlers/members.go:GetMe
  - Uses auth.MemberID + auth.ChurchID

- PUT /members/me → handlers/members.go:UpdateMe
  - Delegates to updateMember(auth.MemberID, auth.ChurchID)

- GET /members/me/instruments → handlers/members.go:GetMyInstruments

- POST /members/me/instruments → handlers/members.go:AddMyInstrument
  - 409 INSTRUMENT_ALREADY_ADDED if duplicate; first instrument auto-set is_primary

- DELETE /members/me/instruments/{instrument_id} → handlers/members.go:RemoveMyInstrument
  - 404 if not in profile

- GET /members/{id} → handlers/members.go:GetMemberByID _(leadership+)_
- PUT /members/{id} → handlers/members.go:UpdateMemberByID _(leadership+)_
- DELETE /members/{id} → handlers/members.go:DeactivateMember _(pastor)_
  - Sets is_active = false; record kept

- GET /members/{id}/roles → handlers/members.go:GetMemberRoles _(leadership+)_
- POST /members/{id}/roles → handlers/members.go:AssignRole _(leadership+)_
  - 403 ROLE_ACCESS_DENIED if role not in church; 409 ROLE_ALREADY_ASSIGNED

- DELETE /members/{id}/roles/{role_id} → handlers/members.go:RemoveMemberRole _(leadership+)_

- GET /members/{id}/instruments → handlers/members.go:GetMemberInstruments _(leadership+)_
  - _(not in architecture endpoint table — added during implementation)_
- POST /members/{id}/instruments → handlers/members.go:AddMemberInstrument _(leadership+)_
- DELETE /members/{id}/instruments/{instrument_id} → handlers/members.go:RemoveMemberInstrument _(leadership+)_

Service: services/member_service.go:MemberService
  - Deps: MemberRepository, RoleRepository, InstrumentRepository, Mailer (nil until resend impl)
  - Error sentinels: ErrMemberEmailExists, ErrMemberNotFound, ErrRoleNotFound,
    ErrRoleAlreadyAssigned, ErrRoleNotAssigned, ErrRoleAccessDenied, ErrSystemResource,
    ErrInstrumentNotFound, ErrInstrumentAlreadyAdded, ErrInstrumentNotInProfile

Repo: adapters/postgres/member_repo.go:MemberRepo
  - Satisfies: MemberRepository + RoleRepository + InstrumentRepository (one struct)
  - Multi-tenant isolation via member_church_memberships JOIN

---

## Roles

- GET /roles → handlers/roles.go:ListRoles _(leadership+)_
  - Returns system roles (church_id IS NULL) + church custom roles

- POST /roles → handlers/roles.go:CreateRole _(pastor)_
  - base_profile must be one of: pastor, leadership, musician, member
  - 409 ROLE_NAME_EXISTS on duplicate name

- PUT /roles/{id} → handlers/roles.go:UpdateRole _(pastor)_
  - 403 SYSTEM_RESOURCE if is_system = true

- DELETE /roles/{id} → handlers/roles.go:DeleteRole _(pastor)_
  - 403 SYSTEM_RESOURCE if is_system = true

---

## Instruments

- GET /instruments → handlers/instruments.go:ListInstruments
  - Returns system instruments (church_id IS NULL) + church custom

- POST /instruments → handlers/instruments.go:CreateInstrument _(leadership+)_
  - 409 INSTRUMENT_NAME_EXISTS on duplicate name

- DELETE /instruments/{id} → handlers/instruments.go:DeleteInstrument _(leadership+)_
  - 403 SYSTEM_RESOURCE if is_system = true

---

## Inventory — Categories

- GET /inventory/categories → handlers/inventory.go:ListCategories
- POST /inventory/categories → handlers/inventory.go:CreateCategory _(leadership+)_
  - 409 CATEGORY_NAME_EXISTS
- PUT /inventory/categories/{id} → handlers/inventory.go:UpdateCategory _(leadership+)_
- DELETE /inventory/categories/{id} → handlers/inventory.go:DeleteCategory _(leadership+)_

---

## Inventory — Items

- GET /inventory/items → handlers/inventory.go:ListItems
  - Filters: search, category_id, status, item_type, include_deleted
  - include_deleted only applied for leadership+ (in-handler check via profileLevel())

- POST /inventory/items → handlers/inventory.go:CreateItem _(leadership+)_
  - Asset number auto-generated as PREFIX-NNN when omitted (services/inventory_service.go:assetPrefix)
  - Prefix = first 4 alpha chars of category name, uppercased; "ITEM" if no category
  - 409 ASSET_NUMBER_EXISTS on duplicate asset_number

- GET /inventory/items/{id} → handlers/inventory.go:GetItemByID

- PUT /inventory/items/{id} → handlers/inventory.go:UpdateItem _(leadership+)_

- POST /inventory/items/{id}/photo → handlers/inventory.go:UploadPhoto _(leadership+)_
  - multipart/form-data; field name = "photo"; max 5 MB enforced
  - R2 integration deferred: currently stores placeholder URL (r2.placeholder/…)
  - Updates items.photo_url

- POST /inventory/items/{id}/discard → handlers/inventory.go:DiscardItem _(leadership+)_
  - Calls DisposeItem(reason="discarded")
  - 409 ITEM_ALREADY_DELETED if deleted_at already set

- POST /inventory/items/{id}/donate → handlers/inventory.go:DonateItem _(leadership+)_
  - Calls DisposeItem(reason="donated"); same 409 guard

Service: services/inventory_service.go:InventoryService
  - Error sentinels: ErrCategoryNotFound, ErrCategoryNameExists, ErrItemNotFound,
    ErrItemAlreadyDeleted, ErrItemNotAvailable, ErrAssetNumberExists,
    ErrLoanNotFound, ErrLoanInvalidStatus, ErrLoanTargetNotFound

---

## Inventory — Loans

- GET /inventory/loans → handlers/inventory.go:ListLoans _(leadership+)_
  - Filter: status, page, per_page

- POST /inventory/loans → handlers/inventory.go:CreateLoan
  - Validates item exists + status="available" + deleted_at IS NULL
  - Validates loan_to_id: member must belong to church; church must exist
  - Auto-approval: pastor/leadership → status=active + item on_loan in same transaction
  - Others → status=pending

- GET /inventory/loans/{id} → handlers/inventory.go:GetLoanByID _(leadership+)_

- POST /inventory/loans/{id}/approve → handlers/inventory.go:ApproveLoan _(leadership+)_
  - Sets status=active + item.status=on_loan; records approved_by

- POST /inventory/loans/{id}/reject → handlers/inventory.go:RejectLoan _(leadership+)_
  - Sets status=rejected; item stays available

- POST /inventory/loans/{id}/return → handlers/inventory.go:ReturnLoan _(leadership+)_
  - return_condition: good | damaged | lost
  - damaged/lost → item.status=maintenance; good → item.status=available

Repo: adapters/postgres/inventory_repo.go:InventoryRepo
  - Satisfies: InventoryRepository
  - Transactional state transitions for loans (Begin/Rollback/Commit pattern)

---

## Database

- Migration file: db/migrations/0001_initial_schema.sql
  - refresh_tokens table appended to same file (not a separate migration)
- All PKs: UUID v4 via gen_random_uuid()
- All timestamps: TIMESTAMPTZ
- Multi-tenant: church_id on every domain table
- Members have no direct church_id; isolation via member_church_memberships JOIN

## Response types

Defined in handlers/auth.go (shared across all handler files):
  memberResponse, roleSummaryResponse, instrumentResponse (MemberInstrument), churchResponse

Per-domain types in their handler file:
  - handlers/roles.go: roleFullResponse
  - handlers/instruments.go: catalogInstrumentResponse
  - handlers/inventory.go: itemCategoryResponse, itemResponse, loanResponse, loanMemberResponse

Builder functions named buildXxxResponse(*ports.Xxx) → xxxResponse.
