-- name: ListStrategyConfigs :many
SELECT id, variant_id, version, strategy_type, params, overlays, params_hash,
       mode, paper_capital, is_current, notes, created_at
FROM strategy_config
ORDER BY variant_id DESC, version DESC;
