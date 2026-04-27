-- 004_attachments_comments.up.sql
-- Defensible-data extensions to sample results: supporting files, threaded
-- comments, and documented override reasons.
--
-- Attachments are stored in an external blob store (filesystem or S3); the
-- database only records metadata and the storage_key needed to fetch the
-- bytes. Soft delete (deleted_at) preserves the audit trail.
--
-- Comments are append-only by design: a comment is a record, not a thread
-- to moderate.
--
-- override_reason is required on sample_results whose entered value falls
-- outside the active validation_rule range for that parameter. Enforced
-- in handlers (not SQL) because the validation rule lookup is dynamic.

CREATE TABLE attachments (
    id              UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id),
    subject_type    TEXT NOT NULL CHECK (subject_type IN ('sample_result')),
    subject_id      UUID NOT NULL,
    filename        TEXT NOT NULL,
    content_type    TEXT NOT NULL,
    size_bytes      BIGINT NOT NULL CHECK (size_bytes > 0),
    storage_key     TEXT NOT NULL UNIQUE,
    uploaded_by     UUID NOT NULL REFERENCES users(id),
    uploaded_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ,
    deleted_by      UUID REFERENCES users(id)
);

CREATE INDEX idx_attachments_subject
    ON attachments (subject_type, subject_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_attachments_org ON attachments (organization_id);

CREATE TABLE comments (
    id              UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id),
    subject_type    TEXT NOT NULL CHECK (subject_type IN ('sample_result')),
    subject_id      UUID NOT NULL,
    author_id       UUID NOT NULL REFERENCES users(id),
    body            TEXT NOT NULL CHECK (length(body) > 0),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_comments_subject
    ON comments (subject_type, subject_id, created_at DESC);

CREATE INDEX idx_comments_org ON comments (organization_id);

ALTER TABLE sample_results ADD COLUMN override_reason TEXT;
