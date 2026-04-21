-- 003_alerts.up.sql
-- Alerts for compliance exceedances and overdue instrument calibrations.
--
-- A background evaluator creates alerts by scanning sample_results (against
-- effective permit_limits) and instruments (against due_at). The partial
-- unique index deduplicates: an active alert for the same subject cannot
-- be created twice. When dismissed, a new alert can be raised again.

CREATE TABLE alerts (
    id              UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id),
    facility_id     UUID NOT NULL REFERENCES facilities(id),
    type            TEXT NOT NULL CHECK (type IN ('exceedance', 'overdue_calibration')),
    severity        TEXT NOT NULL CHECK (severity IN ('info', 'warning', 'critical')),
    subject_type    TEXT NOT NULL CHECK (subject_type IN ('sample_result', 'instrument')),
    subject_id      UUID NOT NULL,
    message         TEXT NOT NULL,
    details         JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    dismissed_at    TIMESTAMPTZ,
    dismissed_by    UUID REFERENCES users(id)
);

-- Dedupe: at most one active (not dismissed) alert per (facility, type, subject).
CREATE UNIQUE INDEX alerts_active_subject_idx
    ON alerts (facility_id, type, subject_type, subject_id)
    WHERE dismissed_at IS NULL;

CREATE INDEX idx_alerts_facility_active
    ON alerts (facility_id, created_at DESC)
    WHERE dismissed_at IS NULL;

CREATE INDEX idx_alerts_org            ON alerts (organization_id);
CREATE INDEX idx_alerts_type           ON alerts (type, created_at DESC);
