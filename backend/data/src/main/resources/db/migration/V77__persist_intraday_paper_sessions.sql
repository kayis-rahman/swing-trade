CREATE TABLE intraday_paper_sessions (
    run_id VARCHAR(36) PRIMARY KEY,
    symbol VARCHAR(32) NOT NULL,
    session_date DATE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    result_json TEXT NOT NULL
);

CREATE INDEX ix_intraday_paper_sessions_created_at
    ON intraday_paper_sessions (created_at DESC);
