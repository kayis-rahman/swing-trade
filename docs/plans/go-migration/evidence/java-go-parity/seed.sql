-- Rows added on top of the Flyway-seeded watchlist, holidays and app_settings
-- before the Java/Go side-by-side run (see ../java-go-endpoint-parity.md).
INSERT INTO signals(symbol, date, signal_type, confidence_score, reasoning, entry_price, stop_loss, target, risk_reward,
    indicators, warning_flag, generated_at, processed, strategy, sentiment_score, sentiment_reasoning, created_at,
    updated_at, version, strategy_version, params_hash) VALUES
('RELIANCE', '2026-11-13', 'BUY', 0.82, 'Pullback to EMA20 in uptrend', 1385.50, 1342.10, 1472.30, 2.00, '{"rsi":58.2}',
    NULL, '2026-11-13', false, 'PULLBACK_IN_UPTREND', 'POSITIVE', 'Strong results', '2026-11-13 16:05:00',
    '2026-11-13 16:05:00', 0, 1, 'abc123'),
('TCS', '2026-11-13', 'HOLD', 0.40, 'No setup', 4120.00, NULL, NULL, NULL, NULL, 'LOW_VOLUME', '2026-11-13', true,
    'PRICE_ACTION', NULL, NULL, '2026-11-13 16:06:00', '2026-11-13 16:06:00', 0, 1, NULL),
('INFY', '2026-11-12', 'SELL', 0.65, 'Breakdown', 1500.25, 1540.00, 1420.50, 2.0062, NULL, NULL, '2026-11-12', false,
    'PRICE_ACTION', 'NEGATIVE', 'Guidance cut', '2026-11-12 16:05:00', '2026-11-12 16:05:00', 0, 1, NULL);

-- Weekday daily bars for October 2026 (including the 2026-10-02 holiday, which reconcile flags as invalid).
INSERT INTO ohlcv_candles(symbol, date, open_price, high_price, low_price, close_price, volume, adj_close_price,
    created_at, version)
SELECT s, d, 100 + i, 102 + i, 99 + i, 101.5 + i, 100000 + i * 10, 101.5 + i, '2026-11-13 18:00:00', 0
FROM (VALUES ('RELIANCE'), ('TCS')) v(s), generate_series(0, 29) i,
     LATERAL (SELECT DATE '2026-10-01' + i AS d) x
WHERE extract(isodow FROM d) < 6;

UPDATE watchlist SET candle_count = (SELECT count(*) FROM ohlcv_candles c WHERE c.symbol = watchlist.symbol),
    last_synced_at = '2026-11-13 18:00:00' WHERE symbol IN ('RELIANCE', 'TCS');
-- A non-zero fraction exercises Java's trailing-zero trimming of LocalDateTime.
UPDATE watchlist SET last_synced_at = '2026-11-13 18:00:00.5' WHERE symbol = 'TCS';
