-- name: GetHolidayByDate :one
SELECT holiday_date, occasion, holiday_type
FROM nse_holidays
WHERE holiday_date = $1;

-- name: ListHolidaysByDateRange :many
SELECT holiday_date, occasion, holiday_type
FROM nse_holidays
WHERE holiday_date BETWEEN $1 AND $2
ORDER BY holiday_date ASC;

-- name: ListUpcomingHolidays :many
SELECT holiday_date, occasion, holiday_type
FROM nse_holidays
WHERE holiday_date >= $1
ORDER BY holiday_date ASC;
