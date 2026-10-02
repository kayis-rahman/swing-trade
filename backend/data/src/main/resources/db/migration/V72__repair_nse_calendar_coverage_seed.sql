-- Restore the reviewed coverage rows if they were removed from an existing database.
-- Preserve any explicit PENDING state already recorded for a year.
INSERT INTO nse_calendar_coverage (calendar_year, status, source, verified_at)
VALUES (2026, 'VERIFIED', 'application holiday calendar', CURRENT_TIMESTAMP),
       (2027, 'VERIFIED', 'application holiday calendar', CURRENT_TIMESTAMP)
ON CONFLICT (calendar_year) DO NOTHING;
