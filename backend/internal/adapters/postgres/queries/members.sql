-- Members queries — input to sqlc generate.
-- Run: sqlc generate
-- Output goes to internal/adapters/postgres/generated/ — DO NOT edit manually.

-- name: ListMembers :many
SELECT m.id, m.name, m.email, m.phone, m.birth_date, m.avatar_url, m.is_active, m.created_at
FROM members m
JOIN member_church_memberships mcm ON m.id = mcm.member_id
WHERE mcm.church_id = $1
  AND mcm.left_at IS NULL
ORDER BY m.name
LIMIT $2 OFFSET $3;

-- name: CountMembers :one
SELECT COUNT(*) FROM members m
JOIN member_church_memberships mcm ON m.id = mcm.member_id
WHERE mcm.church_id = $1 AND mcm.left_at IS NULL;

-- name: CreateMember :one
INSERT INTO members (name, email, phone, birth_date, password_hash)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, name, email, phone, birth_date, avatar_url, is_active, created_at;

-- name: CreateMembership :one
INSERT INTO member_church_memberships (member_id, church_id, is_primary)
VALUES ($1, $2, TRUE)
RETURNING id;

-- name: GetMemberByID :one
SELECT m.id, m.name, m.email, m.phone, m.birth_date, m.avatar_url, m.is_active, m.created_at
FROM members m
JOIN member_church_memberships mcm ON m.id = mcm.member_id
WHERE m.id = $1 AND mcm.church_id = $2 AND mcm.left_at IS NULL;

-- name: UpdateMember :one
UPDATE members
SET name = $1, phone = $2, birth_date = $3, updated_at = NOW()
WHERE id = $4
RETURNING id, name, email, phone, birth_date, avatar_url, is_active, created_at;

-- name: DeactivateMember :exec
UPDATE members SET is_active = FALSE, updated_at = NOW() WHERE id = $1;

-- name: GetMembershipID :one
SELECT id FROM member_church_memberships
WHERE member_id = $1 AND church_id = $2 AND left_at IS NULL;

-- name: GetMemberRoles :many
SELECT r.id, r.name, r.base_profile
FROM member_role_assignments mra
JOIN roles r ON mra.role_id = r.id
JOIN member_church_memberships mcm ON mra.membership_id = mcm.id
WHERE mcm.member_id = $1 AND mcm.church_id = $2 AND mcm.left_at IS NULL;

-- name: AssignRole :exec
INSERT INTO member_role_assignments (membership_id, role_id, assigned_by)
VALUES ($1, $2, $3);

-- name: RemoveRole :exec
DELETE FROM member_role_assignments mra
USING member_church_memberships mcm
WHERE mra.membership_id = mcm.id
  AND mcm.member_id = $1
  AND mcm.church_id = $2
  AND mra.role_id = $3
  AND mcm.left_at IS NULL;

-- name: GetMemberInstruments :many
SELECT mi.id, mi.instrument_id, i.name AS instrument_name, mi.is_primary
FROM member_instruments mi
JOIN instruments i ON mi.instrument_id = i.id
WHERE mi.member_id = $1;

-- name: AddMemberInstrument :one
INSERT INTO member_instruments (member_id, instrument_id, is_primary)
VALUES ($1, $2, $3)
RETURNING id, instrument_id, is_primary;

-- name: RemoveMemberInstrument :exec
DELETE FROM member_instruments WHERE member_id = $1 AND instrument_id = $2;

-- name: ListRoles :many
SELECT id, church_id, name, base_profile, is_system
FROM roles
WHERE church_id IS NULL OR church_id = $1
ORDER BY is_system DESC, name ASC;

-- name: CreateRole :one
INSERT INTO roles (church_id, name, base_profile, is_system)
VALUES ($1, $2, $3, FALSE)
RETURNING id, church_id, name, base_profile, is_system;

-- name: GetRoleByID :one
SELECT id, church_id, name, base_profile, is_system FROM roles WHERE id = $1;

-- name: UpdateRole :one
UPDATE roles SET name = $1, base_profile = $2
WHERE id = $3 AND church_id = $4 AND is_system = FALSE
RETURNING id, church_id, name, base_profile, is_system;

-- name: DeleteRole :exec
DELETE FROM roles WHERE id = $1 AND church_id = $2 AND is_system = FALSE;

-- name: ListInstruments :many
SELECT id, church_id, name, is_system
FROM instruments
WHERE church_id IS NULL OR church_id = $1
ORDER BY is_system DESC, name ASC;

-- name: CreateInstrument :one
INSERT INTO instruments (church_id, name, is_system)
VALUES ($1, $2, FALSE)
RETURNING id, church_id, name, is_system;

-- name: GetInstrumentByID :one
SELECT id, church_id, name, is_system FROM instruments WHERE id = $1;

-- name: DeleteInstrument :exec
DELETE FROM instruments WHERE id = $1 AND church_id = $2 AND is_system = FALSE;
