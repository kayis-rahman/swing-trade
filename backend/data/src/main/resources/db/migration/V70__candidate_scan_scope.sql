-- Preserve the symbol-universe provenance for candidate scan history.
-- NULL is intentional for rows created before this migration.
ALTER TABLE candidate_scan_runs
    ADD COLUMN IF NOT EXISTS scan_scope VARCHAR(16);
