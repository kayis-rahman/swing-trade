-- sqlc schema subset from backend/data/src/main/resources/db/migration/V1__swing_trade_schema.sql.
CREATE TABLE ohlcv_candles (
    id BIGSERIAL PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL,
    date DATE NOT NULL,
    open_price NUMERIC(15,4),
    high_price NUMERIC(15,4),
    low_price NUMERIC(15,4),
    close_price NUMERIC(15,4),
    volume BIGINT,
    adj_close_price NUMERIC(15,4),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    version INTEGER,
    timeframe VARCHAR(10) NOT NULL DEFAULT 'D',
    bar_time TIME NOT NULL DEFAULT '00:00:00',
    CONSTRAINT uq_ohlcv_candles_symbol_timeframe_date_bar_time UNIQUE (symbol, timeframe, date, bar_time)
);

CREATE TABLE watchlist (
    id BIGSERIAL PRIMARY KEY,
    symbol VARCHAR(10) NOT NULL UNIQUE,
    name TEXT,
    exchange VARCHAR(5) DEFAULT 'NSE',
    is_active BOOLEAN DEFAULT TRUE,
    added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_synced_at TIMESTAMP,
    candle_count INTEGER DEFAULT 0
);

CREATE TABLE app_settings (
    id BIGSERIAL PRIMARY KEY,
    key VARCHAR(64) NOT NULL UNIQUE,
    value TEXT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE nse_holidays (
    id BIGSERIAL PRIMARY KEY,
    holiday_date DATE NOT NULL UNIQUE,
    occasion VARCHAR(128) NOT NULL,
    holiday_type VARCHAR(32) NOT NULL DEFAULT 'FULL' CHECK (holiday_type IN ('FULL', 'PARTIAL')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    version INTEGER
);
