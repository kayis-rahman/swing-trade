-- name: DeleteAllFyersSymbols :exec
DELETE FROM fyers_symbol_master;

-- name: InsertFyersSymbol :exec
INSERT INTO fyers_symbol_master (
    fy_token, fyers_symbol, trading_symbol, name, exchange, segment,
    lot_size, tick_size, isin, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: CountFyersSymbols :one
SELECT COUNT(*)::bigint
FROM fyers_symbol_master;

-- name: SearchFyersSymbols :many
SELECT id, fy_token, fyers_symbol, trading_symbol, name, exchange, segment,
       lot_size, tick_size, isin, updated_at
FROM fyers_symbol_master
WHERE UPPER(trading_symbol) LIKE UPPER($1 || '%')
   OR UPPER(name) LIKE UPPER('%' || $1 || '%')
ORDER BY trading_symbol ASC
LIMIT $2;

-- name: FindFyersSymbolsByTradingSymbol :many
SELECT id, fy_token, fyers_symbol, trading_symbol, name, exchange, segment,
       lot_size, tick_size, isin, updated_at
FROM fyers_symbol_master
WHERE UPPER(trading_symbol) = UPPER($1)
ORDER BY trading_symbol ASC;

-- name: ListFyersTradingSymbols :many
SELECT trading_symbol
FROM fyers_symbol_master
ORDER BY trading_symbol ASC;

-- name: ListFyersSymbolsByExchange :many
SELECT id, fy_token, fyers_symbol, trading_symbol, name, exchange, segment,
       lot_size, tick_size, isin, updated_at
FROM fyers_symbol_master
WHERE UPPER(exchange) = UPPER($1)
ORDER BY trading_symbol ASC;
