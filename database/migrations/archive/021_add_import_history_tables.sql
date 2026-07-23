-- Migration: 021_add_import_history_tables.sql
-- Description: Create import job history tables for the reusable import/export framework
-- Created: 2026-06-30

CREATE TABLE import_jobs (
    id               BIGSERIAL    PRIMARY KEY,
    module           VARCHAR(50)  NOT NULL,
    schema_version   VARCHAR(20)  NOT NULL,
    filename         VARCHAR(255) NOT NULL,
    status           VARCHAR(20)  NOT NULL DEFAULT 'queued',
    total_rows       INT          NOT NULL DEFAULT 0,
    inserted         INT          NOT NULL DEFAULT 0,
    updated          INT          NOT NULL DEFAULT 0,
    skipped          INT          NOT NULL DEFAULT 0,
    error_count      INT          NOT NULL DEFAULT 0,
    progress_pct     INT          NOT NULL DEFAULT 0,
    error_report_path VARCHAR(500),
    started_at       TIMESTAMPTZ,
    completed_at     TIMESTAMPTZ,
    duration_ms      INT,
    user_id          INT          NOT NULL REFERENCES users(id),
    store_id         INT          REFERENCES stores(id),
    cancel_requested BOOLEAN      NOT NULL DEFAULT false,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE import_snapshots (
    id               BIGSERIAL    PRIMARY KEY,
    import_job_id    BIGINT       NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE,
    rows_data        JSONB        NOT NULL,
    schema_snapshot  JSONB        NOT NULL,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE import_rows (
    id               BIGSERIAL    PRIMARY KEY,
    import_job_id    BIGINT       NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE,
    row_number       INT          NOT NULL,
    status           VARCHAR(20)  NOT NULL,
    entity_id        INT,
    old_values       JSONB,
    new_values       JSONB,
    changed_fields   TEXT[],
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE import_errors (
    id               BIGSERIAL    PRIMARY KEY,
    import_job_id    BIGINT       NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE,
    row_number       INT          NOT NULL,
    field            VARCHAR(100),
    value            TEXT,
    reason           TEXT         NOT NULL,
    suggestion       TEXT,
    stage            VARCHAR(30)  NOT NULL,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_import_jobs_module ON import_jobs(module);
CREATE INDEX idx_import_jobs_user   ON import_jobs(user_id);
CREATE INDEX idx_import_jobs_status ON import_jobs(status);
CREATE INDEX idx_import_rows_job    ON import_rows(import_job_id);
CREATE INDEX idx_import_errors_job  ON import_errors(import_job_id);
