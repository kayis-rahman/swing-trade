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

CREATE TABLE fyers_symbol_master (
    id BIGSERIAL PRIMARY KEY,
    fy_token VARCHAR(32) NOT NULL UNIQUE,
    fyers_symbol VARCHAR(64) NOT NULL,
    trading_symbol VARCHAR(32) NOT NULL,
    name TEXT,
    exchange VARCHAR(16) NOT NULL DEFAULT 'NSE',
    segment VARCHAR(16),
    lot_size INTEGER,
    tick_size NUMERIC(10,4),
    isin VARCHAR(13),
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX ux_fyers_symbol_master_symbol ON fyers_symbol_master (fyers_symbol);
CREATE INDEX ix_fyers_symbol_master_trading ON fyers_symbol_master (trading_symbol);

CREATE TABLE nse_calendar_coverage (
    calendar_year INTEGER PRIMARY KEY,
    status VARCHAR(16) NOT NULL CHECK (status IN ('VERIFIED', 'PENDING')),
    source VARCHAR(255),
    verified_at TIMESTAMP
);

CREATE TABLE signals (
    id BIGSERIAL PRIMARY KEY,
    symbol VARCHAR(10) NOT NULL,
    date DATE NOT NULL,
    signal_type VARCHAR(10) NOT NULL,
    confidence_score NUMERIC(5,2),
    reasoning TEXT,
    entry_price NUMERIC(15,4),
    stop_loss NUMERIC(15,4),
    target NUMERIC(15,4),
    risk_reward NUMERIC(5,2),
    indicators VARCHAR(255),
    warning_flag VARCHAR(50),
    generated_at DATE,
    processed BOOLEAN DEFAULT FALSE,
    strategy VARCHAR(40) NOT NULL DEFAULT 'DEFAULT',
    sentiment_score VARCHAR(12),
    sentiment_reasoning TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    version INTEGER,
    strategy_version INTEGER NOT NULL DEFAULT 1,
    params_hash VARCHAR(64)
);

CREATE TABLE strategy_config (
    id BIGSERIAL PRIMARY KEY,
    variant_id VARCHAR(40) NOT NULL,
    version INTEGER NOT NULL,
    strategy_type VARCHAR(30) NOT NULL,
    params JSONB NOT NULL,
    overlays JSONB NOT NULL DEFAULT '{}',
    params_hash CHAR(64) NOT NULL,
    mode VARCHAR(16) NOT NULL DEFAULT 'OFF',
    paper_capital NUMERIC(15,2) NOT NULL DEFAULT 500000,
    is_current BOOLEAN NOT NULL DEFAULT TRUE,
    notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
