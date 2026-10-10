ALTER TABLE candidate_scan_runs
    ADD COLUMN IF NOT EXISTS scan_trigger VARCHAR(16);
