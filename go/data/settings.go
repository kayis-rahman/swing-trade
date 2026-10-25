package data

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/kayis-rahman/swing-trade/go/data/sqlc"
)

const defaultSelectedBroker = "yahoo"

// SettingsReadStore exposes read-only application settings needed by API reads.
type SettingsReadStore interface {
	GetSelectedBroker(context.Context) (string, error)
}

// SettingsReader reads the selected market-data provider from app_settings.
type SettingsReader struct {
	queries *sqlc.Queries
}

// NewSettingsReader constructs a reader on a pgx-compatible database handle.
func NewSettingsReader(db sqlc.DBTX) *SettingsReader {
	return &SettingsReader{queries: sqlc.New(db)}
}

// GetSelectedBroker matches the Java provider's default and invalid-value fallback.
func (r *SettingsReader) GetSelectedBroker(ctx context.Context) (string, error) {
	setting, err := r.queries.GetAppSetting(ctx, "selectedBroker")
	if errors.Is(err, pgx.ErrNoRows) {
		return defaultSelectedBroker, nil
	}
	if err != nil {
		return "", fmt.Errorf("get selected broker setting: %w", err)
	}
	if !setting.Valid || !isSupportedBroker(setting.String) {
		return defaultSelectedBroker, nil
	}
	return setting.String, nil
}

func isSupportedBroker(value string) bool {
	switch value {
	case "yahoo", "fyers", "upstox":
		return true
	default:
		return false
	}
}
