-- Auth queries — input to sqlc generate.
-- Run: sqlc generate
-- Output goes to internal/adapters/postgres/generated/ — DO NOT edit manually.

-- name: GetMemberByEmail :one
SELECT id, name, email, password_hash, phone, birth_date, avatar_url, is_active, created_at
FROM members
WHERE email = $1 AND is_active = TRUE;

-- name: GetMemberByID :one
SELECT id, name, email, password_hash, phone, birth_date, avatar_url, is_active, created_at
FROM members
WHERE id = $1;

-- name: GetMemberPrimaryMembership :one
SELECT mcm.id, mcm.church_id
FROM member_church_memberships mcm
WHERE mcm.member_id = $1 AND mcm.is_primary = TRUE AND mcm.left_at IS NULL;

-- name: GetMemberChurchIDs :many
SELECT church_id
FROM member_church_memberships
WHERE member_id = $1 AND left_at IS NULL;

-- name: GetMemberRolesForMembership :many
SELECT r.id, r.name, r.base_profile
FROM member_role_assignments mra
JOIN roles r ON mra.role_id = r.id
WHERE mra.membership_id = $1;

-- name: GetMemberInstruments :many
SELECT mi.id, mi.instrument_id, i.name AS instrument_name, mi.is_primary
FROM member_instruments mi
JOIN instruments i ON mi.instrument_id = i.id
WHERE mi.member_id = $1;

-- name: GetChurchByID :one
SELECT id, parent_church_id, name, denomination_name, cnpj, address,
       is_autonomous, plan_tier, member_count_cache, created_at
FROM churches
WHERE id = $1;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (member_id, jti, expires_at)
VALUES ($1, $2, $3)
RETURNING id, member_id, jti, expires_at, revoked_at, created_at;

-- name: GetRefreshTokenByJTI :one
SELECT id, member_id, jti, expires_at, revoked_at, created_at
FROM refresh_tokens
WHERE jti = $1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET revoked_at = NOW()
WHERE jti = $1 AND revoked_at IS NULL;

-- name: RevokeAllMemberRefreshTokens :exec
UPDATE refresh_tokens
SET revoked_at = NOW()
WHERE member_id = $1 AND revoked_at IS NULL;
