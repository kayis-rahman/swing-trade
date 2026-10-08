-- ============================================================================
-- Stage 0 intraday data plane: interval/timeframe dimension for OHLCV candles.
--
-- Forward-only: this migration does not modify any existing migration (V1-V74).
-- Reversible: roll back with
--   DELETE FROM ohlcv_candles WHERE timeframe <> 'D';  -- required once intraday rows exist
--   ALTER TABLE ohlcv_candles DROP CONSTRAINT uq_ohlcv_candles_symbol_timeframe_date_bar_time;
--   DROP INDEX idx_ohlcv_timeframe_date;
--   ALTER TABLE ohlcv_candles DROP COLUMN timeframe;
--   ALTER TABLE ohlcv_candles DROP COLUMN bar_time;
--   ALTER TABLE ohlcv_candles ADD CONSTRAINT uq_ohlcv_candles_symbol_date UNIQUE (symbol, date);
-- (a rollback discards the intraday dimension; daily rows are unaffected, but the
--  pre-interval unique constraint cannot be re-added while intraday rows exist).
--
-- Existing daily rows are preserved: the column defaults give every pre-existing
-- row timeframe='D' and bar_time=00:00, so daily identity stays (symbol, date).
-- ============================================================================

ALTER TABLE ohlcv_candles
    ADD COLUMN IF NOT EXISTS timeframe VARCHAR(10) NOT NULL DEFAULT 'D',
    ADD COLUMN IF NOT EXISTS bar_time TIME NOT NULL DEFAULT '00:00:00';

-- Widen candle identity from (symbol, date) to (symbol, timeframe, date, bar_time).
-- The pre-existing unique constraint is replaced, not duplicated. Conflicting
-- duplicate rows (same symbol/date, different values) fail loudly here, exactly
-- as they would have against the V29 constraint.
ALTER TABLE ohlcv_candles DROP CONSTRAINT IF EXISTS uq_ohlcv_candles_symbol_date;

-- ADD CONSTRAINT has no IF NOT EXISTS in PostgreSQL; the DO block keeps the
-- migration safely re-runnable.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'uq_ohlcv_candles_symbol_timeframe_date_bar_time'
    ) THEN
        ALTER TABLE ohlcv_candles
            ADD CONSTRAINT uq_ohlcv_candles_symbol_timeframe_date_bar_time
            UNIQUE (symbol, timeframe, date, bar_time);
    END IF;
END
$$;

-- Serves watchlist-wide intraday queries (all symbols for one session); the
-- unique constraint above already covers symbol-first access patterns.
CREATE INDEX IF NOT EXISTS idx_ohlcv_timeframe_date ON ohlcv_candles (timeframe, date);
