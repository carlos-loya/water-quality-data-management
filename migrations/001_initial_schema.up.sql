-- 001_initial_schema.up.sql
-- Foundation schema for water quality data management platform.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "btree_gist";  -- required for permit_limits date exclusion constraint

-- ============================================================================
-- Multi-tenant root
-- ============================================================================

CREATE TABLE organizations (
    id              UUID PRIMARY KEY,
    name            TEXT NOT NULL,
    slug            TEXT NOT NULL UNIQUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================================
-- Users and access control
-- ============================================================================

CREATE TABLE users (
    id              UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id),
    email           TEXT NOT NULL,
    name            TEXT NOT NULL,
    password_hash   TEXT NOT NULL,
    active          BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(organization_id, email)
);

CREATE TABLE roles (
    id              UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id),
    name            TEXT NOT NULL,
    description     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(organization_id, name)
);

-- ============================================================================
-- Reference data
-- ============================================================================

CREATE TABLE units_of_measure (
    id              UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id),
    code            TEXT NOT NULL,   -- 'mg/L', 'NTU', 'SU', 'CFU/100mL'
    name            TEXT NOT NULL,   -- 'milligrams per liter'
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(organization_id, code)
);

CREATE TABLE parameters (
    id              UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id),
    code            TEXT NOT NULL,   -- 'PH', 'CL2', 'TURB', 'BOD5', 'TSS'
    name            TEXT NOT NULL,   -- 'pH', 'Free Chlorine', 'Turbidity'
    description     TEXT,
    cas_number      TEXT,            -- Chemical Abstract Service registry number
    default_unit_id UUID REFERENCES units_of_measure(id),
    category        TEXT,            -- 'physical', 'chemical', 'biological', 'microbiological'
    active          BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(organization_id, code)
);

CREATE TABLE analytical_methods (
    id                  UUID PRIMARY KEY,
    organization_id     UUID NOT NULL REFERENCES organizations(id),
    code                TEXT NOT NULL,   -- 'EPA 150.1', 'SM 4500-Cl G'
    name                TEXT NOT NULL,
    description         TEXT,
    regulatory_reference TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(organization_id, code)
);

-- ============================================================================
-- Facilities and monitoring locations
-- ============================================================================

CREATE TABLE facilities (
    id              UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id),
    name            TEXT NOT NULL,
    facility_type   TEXT NOT NULL,    -- 'water_treatment', 'wastewater_treatment'
    address         TEXT,
    latitude        DOUBLE PRECISION,
    longitude       DOUBLE PRECISION,
    active          BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(organization_id, name)
);

CREATE TABLE monitoring_locations (
    id              UUID PRIMARY KEY,
    facility_id     UUID NOT NULL REFERENCES facilities(id),
    name            TEXT NOT NULL,
    description     TEXT,
    location_type   TEXT,            -- 'influent', 'effluent', 'intermediate', 'distribution'
    latitude        DOUBLE PRECISION,
    longitude       DOUBLE PRECISION,
    active          BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(facility_id, name)
);

-- ============================================================================
-- Compliance: effective-dated permit limits
--
-- The EXCLUDE constraint prevents overlapping date ranges for the same
-- location/parameter/limit_type combination. This guarantees that at any
-- point in time there is at most one active limit, which is critical for
-- defensible compliance evaluations.
-- ============================================================================

CREATE TABLE permit_limits (
    id                      UUID PRIMARY KEY,
    monitoring_location_id  UUID NOT NULL REFERENCES monitoring_locations(id),
    parameter_id            UUID NOT NULL REFERENCES parameters(id),
    unit_id                 UUID NOT NULL REFERENCES units_of_measure(id),
    limit_type              TEXT NOT NULL,        -- 'daily_max', 'daily_min', 'monthly_avg', 'weekly_avg', 'instantaneous_max'
    limit_value             DOUBLE PRECISION NOT NULL,
    effective_start         DATE NOT NULL,
    effective_end           DATE,                 -- NULL = currently active
    permit_reference        TEXT,                 -- permit number or document reference
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    EXCLUDE USING gist (
        monitoring_location_id WITH =,
        parameter_id WITH =,
        limit_type WITH =,
        daterange(effective_start, effective_end, '[]') WITH &&
    )
);

-- ============================================================================
-- Sample results (core operational data)
--
-- This is the highest-volume table. TimescaleDB hypertable partitioned on
-- collected_at for optimized time-series queries and retention policies.
-- ============================================================================

CREATE TABLE sample_results (
    id                      UUID NOT NULL,
    monitoring_location_id  UUID NOT NULL REFERENCES monitoring_locations(id),
    parameter_id            UUID NOT NULL REFERENCES parameters(id),
    method_id               UUID REFERENCES analytical_methods(id),
    unit_id                 UUID NOT NULL REFERENCES units_of_measure(id),
    collected_at            TIMESTAMPTZ NOT NULL,
    analyzed_at             TIMESTAMPTZ,
    result_value            DOUBLE PRECISION,
    result_qualifier        TEXT,                 -- '<', '>', 'ND' (non-detect), NULL for normal
    detection_limit         DOUBLE PRECISION,
    status                  TEXT NOT NULL DEFAULT 'draft',  -- 'draft', 'reviewed', 'approved'
    entered_by              UUID NOT NULL REFERENCES users(id),
    entered_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    reviewed_by             UUID REFERENCES users(id),
    reviewed_at             TIMESTAMPTZ,
    approved_by             UUID REFERENCES users(id),
    approved_at             TIMESTAMPTZ,
    source                  TEXT NOT NULL DEFAULT 'manual', -- 'manual', 'csv_import', 'lims_import', 'api'
    source_reference        TEXT,                 -- import batch ID, LIMS sample ID, etc.
    notes                   TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (id, collected_at)  -- TimescaleDB requires partition column in PK
);

SELECT create_hypertable('sample_results', 'collected_at');

-- ============================================================================
-- Instruments and calibration
-- ============================================================================

CREATE TABLE instruments (
    id                  UUID PRIMARY KEY,
    facility_id         UUID NOT NULL REFERENCES facilities(id),
    name                TEXT NOT NULL,
    serial_number       TEXT,
    instrument_type     TEXT NOT NULL,    -- 'benchtop', 'portable', 'online'
    manufacturer        TEXT,
    model               TEXT,
    location_description TEXT,            -- where in the facility this instrument lives
    active              BOOLEAN NOT NULL DEFAULT true,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE calibration_records (
    id                  UUID PRIMARY KEY,
    instrument_id       UUID NOT NULL REFERENCES instruments(id),
    calibration_type    TEXT NOT NULL,    -- 'verification', 'calibration'
    performed_at        TIMESTAMPTZ NOT NULL,
    performed_by        UUID NOT NULL REFERENCES users(id),
    due_at              TIMESTAMPTZ,     -- next due date
    status              TEXT NOT NULL,    -- 'pass', 'fail'
    pre_value           DOUBLE PRECISION,
    post_value          DOUBLE PRECISION,
    method_reference    TEXT,
    corrective_action   TEXT,
    notes               TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================================
-- User roles (defined after facilities for FK)
--
-- facility_id is nullable: NULL means the role applies to all facilities
-- within the organization.
-- ============================================================================

CREATE TABLE user_roles (
    id              UUID PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users(id),
    role_id         UUID NOT NULL REFERENCES roles(id),
    facility_id     UUID REFERENCES facilities(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX user_roles_unique_idx
    ON user_roles (user_id, role_id, COALESCE(facility_id, '00000000-0000-0000-0000-000000000000'));

-- ============================================================================
-- Audit log
-- ============================================================================

CREATE TABLE audit_log (
    id              UUID NOT NULL,
    organization_id UUID NOT NULL,     -- deliberately no FK: audit records must survive any deletion
    table_name      TEXT NOT NULL,
    record_id       UUID NOT NULL,
    action          TEXT NOT NULL,      -- 'insert', 'update', 'delete'
    old_values      JSONB,
    new_values      JSONB,
    changed_by      UUID NOT NULL,     -- deliberately no FK: audit records must survive user changes
    changed_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    reason          TEXT,
    PRIMARY KEY (id, changed_at)
);

SELECT create_hypertable('audit_log', 'changed_at');

-- ============================================================================
-- Indexes for common query patterns
-- ============================================================================

CREATE INDEX idx_sample_results_location   ON sample_results (monitoring_location_id, collected_at DESC);
CREATE INDEX idx_sample_results_parameter  ON sample_results (parameter_id, collected_at DESC);
CREATE INDEX idx_sample_results_status     ON sample_results (status) WHERE status != 'approved';
CREATE INDEX idx_sample_results_source     ON sample_results (source, source_reference) WHERE source != 'manual';

CREATE INDEX idx_permit_limits_lookup      ON permit_limits (monitoring_location_id, parameter_id, limit_type);
CREATE INDEX idx_calibration_due           ON calibration_records (due_at) WHERE due_at IS NOT NULL;
CREATE INDEX idx_audit_log_record          ON audit_log (table_name, record_id, changed_at DESC);

CREATE INDEX idx_facilities_org            ON facilities (organization_id);
CREATE INDEX idx_monitoring_locations_fac  ON monitoring_locations (facility_id);
CREATE INDEX idx_users_org                 ON users (organization_id);
