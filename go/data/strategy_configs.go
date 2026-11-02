package data

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kayis-rahman/swing-trade/go/data/sqlc"
)

// StrategyConfigEntry is the Java-compatible persisted strategy read model.
type StrategyConfigEntry struct {
	ID           int64           `json:"id"`
	VariantID    string          `json:"variantId"`
	Version      int32           `json:"version"`
	StrategyType string          `json:"strategyType"`
	Params       json.RawMessage `json:"params"`
	Overlays     json.RawMessage `json:"overlays"`
	ParamsHash   string          `json:"paramsHash"`
	Mode         string          `json:"mode"`
	PaperCapital json.Number     `json:"paperCapital"`
	Current      bool            `json:"current"`
	Notes        *string         `json:"notes"`
	CreatedAt    string          `json:"createdAt"`
}

// StrategyConfigReadStore exposes persisted strategy configurations.
type StrategyConfigReadStore interface {
	ListStrategyConfigs(context.Context) ([]StrategyConfigEntry, error)
}

// StrategyConfigReader reads persisted strategy configurations.
type StrategyConfigReader struct{ queries *sqlc.Queries }

// NewStrategyConfigReader constructs a strategy configuration reader.
func NewStrategyConfigReader(db sqlc.DBTX) *StrategyConfigReader {
	return &StrategyConfigReader{queries: sqlc.New(db)}
}

// ListStrategyConfigs returns all versions in Java's descending order.
func (r *StrategyConfigReader) ListStrategyConfigs(ctx context.Context) ([]StrategyConfigEntry, error) {
	rows, err := r.queries.ListStrategyConfigs(ctx)
	if err != nil {
		return nil, fmt.Errorf("list strategy configs: %w", err)
	}
	result := make([]StrategyConfigEntry, 0, len(rows))
	for _, row := range rows {
		capital := fromNumeric(row.PaperCapital)
		if capital == nil {
			return nil, fmt.Errorf("strategy config has invalid paper capital")
		}
		result = append(result, StrategyConfigEntry{
			ID: row.ID, VariantID: row.VariantID, Version: row.Version, StrategyType: row.StrategyType,
			Params: row.Params, Overlays: row.Overlays, ParamsHash: row.ParamsHash, Mode: row.Mode,
			PaperCapital: json.Number(*capital), Current: row.IsCurrent, Notes: nullableText(row.Notes),
			CreatedAt: row.CreatedAt.Time.Format("2006-01-02T15:04:05.000000"),
		})
	}
	return result, nil
}
