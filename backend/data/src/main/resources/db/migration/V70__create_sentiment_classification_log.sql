-- Per-classification-call log for the Laya local sentiment pre-filter and the
-- Qwen/LLM sentiment stage, used to compute Laya/Qwen agreement rate during
-- Laya's shadow-mode rollout. One row per model call (so a single
-- SentimentService.performSentimentAnalysis invocation in shadow mode writes
-- two rows: one LAYA, one QWEN, for the same symbol/analysis_date).
CREATE TABLE IF NOT EXISTS sentiment_classification_log (
    id BIGSERIAL PRIMARY KEY,
    version INTEGER NOT NULL DEFAULT 0,
    symbol VARCHAR(20) NOT NULL,
    analysis_date DATE NOT NULL,
    model_used VARCHAR(10) NOT NULL,
    sentiment VARCHAR(10) NOT NULL,
    confidence DOUBLE PRECISION,
    latency_ms BIGINT,
    shadow_mode BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sentiment_classification_log_symbol_date
    ON sentiment_classification_log (symbol, analysis_date);
CREATE INDEX IF NOT EXISTS idx_sentiment_classification_log_model
    ON sentiment_classification_log (model_used);
CREATE INDEX IF NOT EXISTS idx_sentiment_classification_log_created
    ON sentiment_classification_log (created_at);
