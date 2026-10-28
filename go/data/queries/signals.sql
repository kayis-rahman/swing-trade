-- name: ListLatestSignals :many
SELECT id, symbol, date, signal_type, confidence_score, reasoning, entry_price, stop_loss, target,
       risk_reward, indicators, warning_flag, generated_at, processed, strategy, sentiment_score,
       sentiment_reasoning, created_at, updated_at, version, strategy_version, params_hash
FROM (
    SELECT s.*,
           ROW_NUMBER() OVER (
               PARTITION BY s.symbol ORDER BY s.date DESC, s.created_at DESC, s.id DESC
           ) AS rn
    FROM signals s
) ranked
WHERE ranked.rn = 1
ORDER BY ranked.symbol ASC;
