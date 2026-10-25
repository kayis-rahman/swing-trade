-- name: GetEarliestDailyCandleDate :one
SELECT date
FROM ohlcv_candles
WHERE symbol = $1 AND timeframe = 'D'
ORDER BY date ASC
LIMIT 1;

-- name: GetLatestDailyCandleDate :one
SELECT date
FROM ohlcv_candles
WHERE symbol = $1 AND timeframe = 'D'
ORDER BY date DESC
LIMIT 1;

-- name: CountDailyCandles :one
SELECT COUNT(*)::bigint
FROM ohlcv_candles
WHERE symbol = $1 AND timeframe = 'D';
