-- name: GetAppSetting :one
SELECT value
FROM app_settings
WHERE key = $1;
