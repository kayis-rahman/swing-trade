-- name: ListActiveWatchlist :many
SELECT id, symbol, name, exchange, is_active, added_at, last_synced_at, candle_count
FROM watchlist
WHERE is_active = TRUE
ORDER BY symbol ASC;

-- name: ListAllWatchlist :many
SELECT id, symbol, name, exchange, is_active, added_at, last_synced_at, candle_count
FROM watchlist
ORDER BY id ASC;
