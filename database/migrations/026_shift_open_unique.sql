-- Migration: 026_shift_open_unique.sql
-- Description: Prevent concurrent open shifts per user with a partial unique
-- index on open shifts. The app-level pre-check (repository.go OpenShift) is
-- kept as a fast path; this index is the authoritative guard against the
-- SELECT-then-INSERT race.
--
-- Guard (conflict #6 in the security remediation plan): the schema previously
-- allowed a user to hold multiple open shifts. If any already exist at apply
-- time, this migration either:
--   * fails loudly (default) -- no open shifts are silently lost; the operator
--     must resolve duplicates by hand, or
--   * auto-closes the older duplicate open shifts if the session GUC
--     app.shift_migration_mode is set to 'auto-close' (dev/dummy-data only;
--     dummy data is regenerable).
--
-- Set the mode from a shell/env-driven runner, e.g.
--   psql -c "SET app.shift_migration_mode = 'auto-close'" -d retail_pos < this file
-- or via PGOPTIONS="-c app.shift_migration_mode=auto-close".

DO $$
DECLARE
    dup_count  BIGINT;
    mode       TEXT := COALESCE(current_setting('app.shift_migration_mode', true), 'fail');
BEGIN
    IF EXISTS (SELECT 1 FROM pg_catalog.pg_indexes WHERE indexname = 'uq_open_shift_per_user') THEN
        RETURN;
    END IF;

    SELECT COUNT(*) INTO dup_count FROM (
        SELECT user_id
        FROM shifts
        WHERE status = 'open'
        GROUP BY user_id
        HAVING COUNT(*) > 1
    ) d;

    IF dup_count > 0 THEN
        IF mode = 'auto-close' THEN
            UPDATE shifts
            SET status = 'closed',
                closed_at = NOW(),
                updated_at = NOW(),
                notes = COALESCE(notes, '') || E'\n[026] Auto-closed duplicate open shift by migration.'
            WHERE id IN (
                SELECT id FROM (
                    SELECT id,
                           ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY opened_at DESC, id DESC) AS rn
                    FROM shifts
                    WHERE status = 'open'
                ) ranked
                WHERE ranked.rn > 1
            );
        ELSE
            RAISE EXCEPTION
                'uq_open_shift_per_user cannot be created: % user(s) currently have multiple open shifts. '
                'Resolve the duplicates manually, or in dev/dummy-data environments re-run with '
                'app.shift_migration_mode=auto-close to auto-close the older duplicate open shifts.',
                dup_count;
        END IF;
    END IF;

    CREATE UNIQUE INDEX IF NOT EXISTS uq_open_shift_per_user ON shifts (user_id) WHERE status = 'open';
END
$$;