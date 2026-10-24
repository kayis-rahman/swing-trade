-- sqlc schema subset from backend/data/src/main/resources/db/migration/V1__swing_trade_schema.sql.
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
