-- =============================================================================
-- Igreja Organizada — Database Schema
-- Version: 1.0.0
-- Database: PostgreSQL 15+
-- Encoding: UTF-8
-- =============================================================================
-- Conventions:
--   - All PKs are UUID v4 (gen_random_uuid())
--   - All timestamps are timestamptz (UTC stored, local displayed)
--   - All tables have created_at. Mutable tables also have updated_at.
--   - Multi-tenant isolation via church_id on every domain table
--   - Soft deletes: deleted_at on items only; flags elsewhere
-- =============================================================================

-- Enable UUID generation (built-in on PG 13+)
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- =============================================================================
-- DOMAIN 1 — IDENTITY & ORGANISATION
-- =============================================================================

-- -----------------------------------------------------------------------------
-- churches
-- One row per local unit (matrix or congregation).
-- Billing fields are only relevant on matrix rows (parent_church_id IS NULL).
-- -----------------------------------------------------------------------------
CREATE TABLE churches (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_church_id    UUID        REFERENCES churches(id) ON DELETE RESTRICT,
    name                VARCHAR(200) NOT NULL,
    denomination_name   VARCHAR(200),
    cnpj                VARCHAR(18)  UNIQUE,
    address             TEXT,
    is_autonomous       BOOLEAN     NOT NULL DEFAULT FALSE,
    billing_email       VARCHAR(200),
    plan_tier           VARCHAR(20)  NOT NULL DEFAULT 'free'
                            CHECK (plan_tier IN ('free', 'basic', 'growth', 'enterprise')),
    member_count_cache  INTEGER     NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  churches                  IS 'Local church units. Matrix rows have parent_church_id = NULL.';
COMMENT ON COLUMN churches.is_autonomous    IS 'TRUE = congregation became an independent church (no longer reports to matrix).';
COMMENT ON COLUMN churches.member_count_cache IS 'Denormalised count including all child churches. Updated by nightly job.';

CREATE INDEX idx_churches_parent ON churches(parent_church_id) WHERE parent_church_id IS NOT NULL;


-- -----------------------------------------------------------------------------
-- members
-- One row per person. Church link lives in member_church_memberships.
-- -----------------------------------------------------------------------------
CREATE TABLE members (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(200) NOT NULL,
    email           VARCHAR(200) NOT NULL UNIQUE,
    phone           VARCHAR(20),
    password_hash   TEXT        NOT NULL,
    birth_date      DATE,
    avatar_url      TEXT,                          -- Phase 2
    is_active       BOOLEAN     NOT NULL DEFAULT TRUE,
    last_login_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  members           IS 'One row per person. No church_id here — link is in member_church_memberships.';
COMMENT ON COLUMN members.email     IS 'Used for login. Unique across the entire system.';
COMMENT ON COLUMN members.is_active IS 'FALSE = member left. Record is kept for historical references.';

CREATE INDEX idx_members_email ON members(email);


-- -----------------------------------------------------------------------------
-- roles
-- System roles (is_system = TRUE, church_id = NULL) are seeded and immutable.
-- Churches can create custom roles that inherit from a base_profile.
-- -----------------------------------------------------------------------------
CREATE TABLE roles (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    church_id       UUID        REFERENCES churches(id) ON DELETE CASCADE,
    name            VARCHAR(100) NOT NULL,
    base_profile    VARCHAR(20)  NOT NULL
                        CHECK (base_profile IN ('pastor', 'leadership', 'musician', 'member')),
    is_system       BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_roles_church_name UNIQUE (church_id, name)
);

COMMENT ON TABLE  roles              IS 'System roles (church_id NULL) + custom roles per church.';
COMMENT ON COLUMN roles.base_profile IS 'Permission profile inherited: pastor > leadership > musician > member.';
COMMENT ON COLUMN roles.is_system    IS 'TRUE = seeded by system, cannot be edited or deleted.';

CREATE INDEX idx_roles_church ON roles(church_id) WHERE church_id IS NOT NULL;


-- -----------------------------------------------------------------------------
-- member_church_memberships
-- A member can belong to multiple churches (e.g. presbyter active in both
-- matrix and congregation). is_primary defines the default church on login.
-- -----------------------------------------------------------------------------
CREATE TABLE member_church_memberships (
    id          UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id   UUID    NOT NULL REFERENCES members(id)  ON DELETE CASCADE,
    church_id   UUID    NOT NULL REFERENCES churches(id) ON DELETE CASCADE,
    is_primary  BOOLEAN NOT NULL DEFAULT TRUE,
    joined_at   DATE    NOT NULL DEFAULT CURRENT_DATE,
    left_at     DATE,

    CONSTRAINT uq_membership UNIQUE (member_id, church_id)
);

COMMENT ON TABLE  member_church_memberships          IS 'Link between a member and a church. One member can have multiple memberships.';
COMMENT ON COLUMN member_church_memberships.left_at  IS 'NULL = still active in this church.';

CREATE INDEX idx_memberships_member ON member_church_memberships(member_id);
CREATE INDEX idx_memberships_church ON member_church_memberships(church_id);


-- -----------------------------------------------------------------------------
-- member_role_assignments
-- Which roles a member holds within a specific church membership.
-- -----------------------------------------------------------------------------
CREATE TABLE member_role_assignments (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    membership_id   UUID        NOT NULL REFERENCES member_church_memberships(id) ON DELETE CASCADE,
    role_id         UUID        NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
    assigned_by     UUID        NOT NULL REFERENCES members(id) ON DELETE RESTRICT,
    assigned_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_role_assignment UNIQUE (membership_id, role_id)
);

CREATE INDEX idx_role_assignments_membership ON member_role_assignments(membership_id);


-- =============================================================================
-- DOMAIN 2 — PASTORAL AGENDA
-- =============================================================================

-- -----------------------------------------------------------------------------
-- availability_slots
-- Recurring time windows the pastor opens for bookings (by day of week).
-- -----------------------------------------------------------------------------
CREATE TABLE availability_slots (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    church_id   UUID        NOT NULL REFERENCES churches(id) ON DELETE CASCADE,
    created_by  UUID        NOT NULL REFERENCES members(id)  ON DELETE RESTRICT,
    day_of_week SMALLINT    NOT NULL CHECK (day_of_week BETWEEN 0 AND 6), -- 0=Sun, 6=Sat
    start_time  TIME        NOT NULL,
    end_time    TIME        NOT NULL,
    is_active   BOOLEAN     NOT NULL DEFAULT TRUE,

    CONSTRAINT chk_slot_times CHECK (end_time > start_time)
);

COMMENT ON COLUMN availability_slots.day_of_week IS '0 = Sunday, 1 = Monday, ... 6 = Saturday.';

CREATE INDEX idx_avail_slots_church ON availability_slots(church_id);


-- -----------------------------------------------------------------------------
-- events
-- Every pastoral appointment. Exists from the moment of request.
-- Appears on the calendar only when status = confirmed.
-- -----------------------------------------------------------------------------
CREATE TABLE events (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    church_id       UUID        NOT NULL REFERENCES churches(id) ON DELETE CASCADE,
    title           VARCHAR(200) NOT NULL,
    event_type      VARCHAR(30)  NOT NULL
                        CHECK (event_type IN ('pastoral_visit', 'counseling', 'leadership_meeting', 'block', 'other')),
    status          VARCHAR(20)  NOT NULL DEFAULT 'requested'
                        CHECK (status IN ('requested', 'confirmed', 'declined', 'cancelled')),
    requested_by    UUID        REFERENCES members(id) ON DELETE SET NULL,  -- NULL = pastor created directly
    approved_by     UUID        REFERENCES members(id) ON DELETE SET NULL,
    starts_at       TIMESTAMPTZ NOT NULL,
    ends_at         TIMESTAMPTZ NOT NULL,
    location        TEXT,
    notes           TEXT,                          -- Visible to pastor and leadership only
    decline_reason  TEXT,                          -- Sent by email to requester on decline
    gcal_event_id   VARCHAR(200),                  -- Google Calendar event ID
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_event_times CHECK (ends_at > starts_at)
);

COMMENT ON COLUMN events.requested_by  IS 'NULL = pastor created the event directly (e.g. a block).';
COMMENT ON COLUMN events.gcal_event_id IS 'Stored after creating the Google Calendar invite. Used for updates and cancellations.';

CREATE INDEX idx_events_church        ON events(church_id);
CREATE INDEX idx_events_status        ON events(status);
CREATE INDEX idx_events_starts_at     ON events(starts_at);
CREATE INDEX idx_events_requested_by  ON events(requested_by) WHERE requested_by IS NOT NULL;


-- -----------------------------------------------------------------------------
-- event_attendees
-- Participants of an event. For pastoral visits usually one person;
-- for leadership meetings can be many.
-- -----------------------------------------------------------------------------
CREATE TABLE event_attendees (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id        UUID        NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    member_id       UUID        NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    status          VARCHAR(20)  NOT NULL DEFAULT 'invited'
                        CHECK (status IN ('invited', 'accepted', 'declined', 'attended')),
    responded_at    TIMESTAMPTZ,

    CONSTRAINT uq_event_attendee UNIQUE (event_id, member_id)
);

CREATE INDEX idx_event_attendees_event  ON event_attendees(event_id);
CREATE INDEX idx_event_attendees_member ON event_attendees(member_id);


-- =============================================================================
-- DOMAIN 3 — WORSHIP SCHEDULE
-- =============================================================================

-- -----------------------------------------------------------------------------
-- instruments
-- Master list of instruments / musical functions per church.
-- Seeded with defaults; churches can add their own.
-- -----------------------------------------------------------------------------
CREATE TABLE instruments (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    church_id   UUID        REFERENCES churches(id) ON DELETE CASCADE,  -- NULL = system default
    name        VARCHAR(100) NOT NULL,
    is_system   BOOLEAN     NOT NULL DEFAULT FALSE,

    CONSTRAINT uq_instrument_church_name UNIQUE (church_id, name)
);

COMMENT ON TABLE  instruments           IS 'Musical instruments and vocal functions. System defaults have church_id = NULL.';
COMMENT ON COLUMN instruments.is_system IS 'TRUE = seeded by system, cannot be deleted.';

CREATE INDEX idx_instruments_church ON instruments(church_id) WHERE church_id IS NOT NULL;


-- -----------------------------------------------------------------------------
-- member_instruments
-- A musician can play multiple instruments / sing.
-- -----------------------------------------------------------------------------
CREATE TABLE member_instruments (
    id              UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id       UUID    NOT NULL REFERENCES members(id)     ON DELETE CASCADE,
    instrument_id   UUID    NOT NULL REFERENCES instruments(id) ON DELETE RESTRICT,
    is_primary      BOOLEAN NOT NULL DEFAULT FALSE,  -- Primary instrument for auto-scheduling

    CONSTRAINT uq_member_instrument UNIQUE (member_id, instrument_id)
);

COMMENT ON COLUMN member_instruments.is_primary IS 'TRUE = preferred instrument used as hint for schedule suggestion.';

CREATE INDEX idx_member_instruments_member     ON member_instruments(member_id);
CREATE INDEX idx_member_instruments_instrument ON member_instruments(instrument_id);


-- -----------------------------------------------------------------------------
-- availability_exceptions
-- Days a musician CANNOT play. System assumes available by default.
-- Only exceptions are stored.
-- -----------------------------------------------------------------------------
CREATE TABLE availability_exceptions (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id           UUID        NOT NULL REFERENCES members(id)  ON DELETE CASCADE,
    church_id           UUID        NOT NULL REFERENCES churches(id) ON DELETE CASCADE,
    unavailable_date    DATE        NOT NULL,
    reason              VARCHAR(200),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_avail_exception UNIQUE (member_id, church_id, unavailable_date)
);

COMMENT ON TABLE  availability_exceptions                  IS 'Days a musician cannot play. Available = default assumption.';
COMMENT ON COLUMN availability_exceptions.unavailable_date IS 'The specific Sunday the member cannot play.';

CREATE INDEX idx_avail_exceptions_member ON availability_exceptions(member_id);
CREATE INDEX idx_avail_exceptions_church ON availability_exceptions(church_id);
CREATE INDEX idx_avail_exceptions_date   ON availability_exceptions(unavailable_date);


-- -----------------------------------------------------------------------------
-- schedules
-- The worship lineup for one specific Sunday. One per Sunday per church.
-- -----------------------------------------------------------------------------
CREATE TABLE schedules (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    church_id       UUID        NOT NULL REFERENCES churches(id) ON DELETE CASCADE,
    sunday_date     DATE        NOT NULL,
    status          VARCHAR(20)  NOT NULL DEFAULT 'draft'
                        CHECK (status IN ('draft', 'published', 'cancelled')),
    created_by      UUID        NOT NULL REFERENCES members(id) ON DELETE RESTRICT,
    approved_by     UUID        REFERENCES members(id) ON DELETE SET NULL,
    notes           TEXT,
    published_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_schedule_sunday UNIQUE (church_id, sunday_date)
);

COMMENT ON COLUMN schedules.sunday_date IS 'Must be a Sunday. Enforced in application layer.';

CREATE INDEX idx_schedules_church ON schedules(church_id);
CREATE INDEX idx_schedules_date   ON schedules(sunday_date);


-- -----------------------------------------------------------------------------
-- schedule_slots
-- Each row in the lineup: who plays what that Sunday.
-- function_in_scale is a FK to instruments instead of free text,
-- enabling future auto-scheduling by instrument.
-- -----------------------------------------------------------------------------
CREATE TABLE schedule_slots (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    schedule_id         UUID        NOT NULL REFERENCES schedules(id)    ON DELETE CASCADE,
    member_id           UUID        NOT NULL REFERENCES members(id)      ON DELETE RESTRICT,
    instrument_id       UUID        REFERENCES instruments(id)           ON DELETE SET NULL,
    function_in_scale   VARCHAR(100) NOT NULL,  -- Display label (kept for flexibility)
    confirmed           BOOLEAN     NOT NULL DEFAULT FALSE,
    notified_at         TIMESTAMPTZ,

    CONSTRAINT uq_slot_member_schedule UNIQUE (schedule_id, member_id)
);

COMMENT ON COLUMN schedule_slots.instrument_id     IS 'FK to instruments for future auto-scheduling. Mirrors function_in_scale.';
COMMENT ON COLUMN schedule_slots.function_in_scale IS 'Display label e.g. "Guitar", "Lead Vocal". Populated from instrument.name.';

CREATE INDEX idx_schedule_slots_schedule ON schedule_slots(schedule_id);
CREATE INDEX idx_schedule_slots_member   ON schedule_slots(member_id);


-- =============================================================================
-- DOMAIN 4 — INVENTORY
-- =============================================================================

-- -----------------------------------------------------------------------------
-- item_categories
-- Customisable per church. e.g. Instruments, AV Equipment, Furniture.
-- -----------------------------------------------------------------------------
CREATE TABLE item_categories (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    church_id   UUID        NOT NULL REFERENCES churches(id) ON DELETE CASCADE,
    name        VARCHAR(100) NOT NULL,
    icon        VARCHAR(50),

    CONSTRAINT uq_category_church_name UNIQUE (church_id, name)
);

CREATE INDEX idx_item_categories_church ON item_categories(church_id);


-- -----------------------------------------------------------------------------
-- items
-- One row per physical item.
-- asset:      individual patrimony number, photo, full loan history.
-- consumable: quantity + min-alert threshold, no individual loans.
-- Soft delete via deleted_at + deletion_reason.
-- -----------------------------------------------------------------------------
CREATE TABLE items (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    church_id           UUID        NOT NULL REFERENCES churches(id)        ON DELETE RESTRICT,
    category_id         UUID        REFERENCES item_categories(id)          ON DELETE SET NULL,
    item_type           VARCHAR(15)  NOT NULL
                            CHECK (item_type IN ('asset', 'consumable')),
    name                VARCHAR(200) NOT NULL,
    description         TEXT,
    asset_number        VARCHAR(50),               -- Assets only; auto-generated or manual
    photo_url           TEXT,                      -- Cloudflare R2 URL
    location            VARCHAR(100) NOT NULL DEFAULT 'Main Hall',
    status              VARCHAR(20)  NOT NULL DEFAULT 'available'
                            CHECK (status IN ('available', 'on_loan', 'maintenance')),
    quantity            INTEGER     NOT NULL DEFAULT 1 CHECK (quantity >= 0),
    qty_min_alert       INTEGER     CHECK (qty_min_alert >= 0),  -- Consumables only
    serial_number       VARCHAR(100),
    notes               TEXT,
    deleted_at          TIMESTAMPTZ,               -- Soft delete
    deletion_reason     VARCHAR(20)
                            CHECK (deletion_reason IN ('discarded', 'donated')),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_asset_number UNIQUE (church_id, asset_number),
    CONSTRAINT chk_asset_number_required
        CHECK (item_type = 'consumable' OR asset_number IS NOT NULL),
    CONSTRAINT chk_deletion_consistent
        CHECK ((deleted_at IS NULL) = (deletion_reason IS NULL))
);

COMMENT ON COLUMN items.asset_number   IS 'Required for assets. NULL for consumables.';
COMMENT ON COLUMN items.status         IS 'Active status. Does not include deleted items (use deleted_at for those).';
COMMENT ON COLUMN items.deleted_at     IS 'Soft delete. Non-NULL = item was discarded or donated.';
COMMENT ON COLUMN items.deletion_reason IS 'Reason for soft delete: discarded | donated.';

CREATE INDEX idx_items_church      ON items(church_id);
CREATE INDEX idx_items_status      ON items(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_items_active      ON items(church_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_items_category    ON items(category_id) WHERE category_id IS NOT NULL;


-- -----------------------------------------------------------------------------
-- loans
-- Asset loans to another church or to a member.
-- Records are never deleted — status tracking only.
-- Polymorphic target: loan_to_type + loan_to_id resolves to churches or members.
-- -----------------------------------------------------------------------------
CREATE TABLE loans (
    id                      UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    item_id                 UUID        NOT NULL REFERENCES items(id)   ON DELETE RESTRICT,
    requested_by            UUID        NOT NULL REFERENCES members(id) ON DELETE RESTRICT,
    approved_by             UUID        REFERENCES members(id)          ON DELETE SET NULL,
    loan_to_type            VARCHAR(10)  NOT NULL CHECK (loan_to_type IN ('church', 'member')),
    loan_to_id              UUID        NOT NULL,  -- Resolves to churches.id or members.id
    status                  VARCHAR(25)  NOT NULL DEFAULT 'pending'
                                CHECK (status IN ('pending', 'active', 'returned', 'returned_with_issue', 'rejected')),
    expected_return_date    DATE,
    actual_return_date      DATE,
    return_condition        VARCHAR(20)
                                CHECK (return_condition IN ('good', 'damaged', 'lost')),
    return_notes            TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    returned_at             TIMESTAMPTZ,

    CONSTRAINT chk_return_date_consistent
        CHECK (actual_return_date IS NULL OR actual_return_date >= created_at::DATE),
    CONSTRAINT chk_return_condition_when_returned
        CHECK (
            status NOT IN ('returned', 'returned_with_issue')
            OR return_condition IS NOT NULL
        )
);

COMMENT ON COLUMN loans.loan_to_type IS 'church = lent to another church; member = lent to a person.';
COMMENT ON COLUMN loans.loan_to_id   IS 'UUID resolving to churches.id or members.id depending on loan_to_type.';
COMMENT ON COLUMN loans.returned_at  IS 'Timestamp of physical return. actual_return_date is the date portion.';

CREATE INDEX idx_loans_item        ON loans(item_id);
CREATE INDEX idx_loans_requested   ON loans(requested_by);
CREATE INDEX idx_loans_status      ON loans(status);
CREATE INDEX idx_loans_active      ON loans(item_id) WHERE status = 'active';


-- =============================================================================
-- INFRASTRUCTURE
-- =============================================================================

-- -----------------------------------------------------------------------------
-- notifications
-- Append-only log of all notifications sent.
-- Enables resend, diagnosis and audit trail.
-- -----------------------------------------------------------------------------
CREATE TABLE notifications (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id       UUID        NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    channel         VARCHAR(20)  NOT NULL CHECK (channel IN ('email', 'push')),
    template        VARCHAR(50)  NOT NULL,   -- e.g. 'schedule_published', 'loan_overdue'
    reference_type  VARCHAR(30),             -- 'schedule' | 'loan' | 'event'
    reference_id    UUID,                    -- ID of the related object
    status          VARCHAR(20)  NOT NULL DEFAULT 'queued'
                        CHECK (status IN ('queued', 'sent', 'failed')),
    sent_at         TIMESTAMPTZ,
    error_message   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  notifications             IS 'Append-only notification log. Never update or delete rows.';
COMMENT ON COLUMN notifications.template    IS 'Template key used by the email service.';
COMMENT ON COLUMN notifications.reference_id IS 'Polymorphic: the ID of the schedule, loan or event that triggered this notification.';

CREATE INDEX idx_notifications_member ON notifications(member_id);
CREATE INDEX idx_notifications_status ON notifications(status) WHERE status = 'queued';


-- =============================================================================
-- UPDATED_AT TRIGGER
-- Automatically sets updated_at on every UPDATE.
-- =============================================================================

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_churches_updated_at
    BEFORE UPDATE ON churches
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_members_updated_at
    BEFORE UPDATE ON members
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_events_updated_at
    BEFORE UPDATE ON events
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_schedules_updated_at
    BEFORE UPDATE ON schedules
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_items_updated_at
    BEFORE UPDATE ON items
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();


-- =============================================================================
-- SEED DATA
-- =============================================================================

-- -----------------------------------------------------------------------------
-- System roles (church_id = NULL, is_system = TRUE)
-- These cannot be edited or deleted by any church.
-- -----------------------------------------------------------------------------
INSERT INTO roles (id, church_id, name, base_profile, is_system) VALUES
    (gen_random_uuid(), NULL, 'Pastor',              'pastor',     TRUE),
    (gen_random_uuid(), NULL, 'Leadership',          'leadership', TRUE),
    (gen_random_uuid(), NULL, 'Musician',            'musician',   TRUE),
    (gen_random_uuid(), NULL, 'Member',              'member',     TRUE),
    (gen_random_uuid(), NULL, 'Asset Manager',       'leadership', TRUE),
    (gen_random_uuid(), NULL, 'Worship Leader',      'leadership', TRUE);

-- -----------------------------------------------------------------------------
-- System instruments (church_id = NULL, is_system = TRUE)
-- Churches can add their own; these are always available.
-- -----------------------------------------------------------------------------
INSERT INTO instruments (id, church_id, name, is_system) VALUES
    (gen_random_uuid(), NULL, 'Lead Vocal',      TRUE),
    (gen_random_uuid(), NULL, 'Back Vocal',      TRUE),
    (gen_random_uuid(), NULL, 'Acoustic Guitar', TRUE),
    (gen_random_uuid(), NULL, 'Electric Guitar', TRUE),
    (gen_random_uuid(), NULL, 'Bass Guitar',     TRUE),
    (gen_random_uuid(), NULL, 'Keyboard',        TRUE),
    (gen_random_uuid(), NULL, 'Piano',           TRUE),
    (gen_random_uuid(), NULL, 'Drums',           TRUE),
    (gen_random_uuid(), NULL, 'Cajon',           TRUE),
    (gen_random_uuid(), NULL, 'Violin',          TRUE),
    (gen_random_uuid(), NULL, 'Viola',           TRUE),
    (gen_random_uuid(), NULL, 'Cello',           TRUE),
    (gen_random_uuid(), NULL, 'Trumpet',         TRUE),
    (gen_random_uuid(), NULL, 'Saxophone',       TRUE),
    (gen_random_uuid(), NULL, 'Flute',           TRUE),
    (gen_random_uuid(), NULL, 'Sound Engineer',  TRUE),
    (gen_random_uuid(), NULL, 'Media/Slides',    TRUE);