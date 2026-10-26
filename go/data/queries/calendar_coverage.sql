-- name: ListNseCalendarCoverageYears :many
SELECT calendar_year, status
FROM nse_calendar_coverage
WHERE calendar_year BETWEEN $1 AND $2;
