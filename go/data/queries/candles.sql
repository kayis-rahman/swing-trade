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

-- name: DailyCandleExists :one
SELECT EXISTS(
    SELECT 1
    FROM ohlcv_candles
    WHERE symbol = $1 AND timeframe = 'D' AND date = $2
);

-- name: GetLatestIntervalCandleDate :one
SELECT date
FROM ohlcv_candles
WHERE symbol = $1 AND timeframe = $2
ORDER BY date DESC
LIMIT 1;
