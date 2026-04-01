-- Configurable validation rules per parameter.
-- Each parameter can have a valid range (min/max) and precision constraint.
-- These are enforced on sample result entry (both API and import).

CREATE TABLE validation_rules (
    id              UUID PRIMARY KEY,
    parameter_id    UUID NOT NULL REFERENCES parameters(id),
    min_value       DOUBLE PRECISION,       -- NULL means no lower bound
    max_value       DOUBLE PRECISION,       -- NULL means no upper bound
    precision_digits SMALLINT,              -- max decimal places allowed (NULL = no constraint)
    is_required     BOOLEAN NOT NULL DEFAULT false,  -- whether a numeric value (not just qualifier) is required
    active          BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(parameter_id)
);
