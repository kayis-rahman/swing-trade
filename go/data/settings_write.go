package data

import (
	"context"
	"fmt"
	"strings"

	"github.com/kayis-rahman/swing-trade/go/data/sqlc"
)

// SettingsWriteStore persists application settings.
type SettingsWriteStore interface {
	SetSetting(ctx context.Context, key, value string) error
}

// SettingsWriter writes application settings to app_settings with an upsert.
type SettingsWriter struct {
	db sqlc.DBTX
}

// NewSettingsWriter constructs a writer against a pgx-compatible database handle.
func NewSettingsWriter(db sqlc.DBTX) (*SettingsWriter, error) {
	if db == nil {
		return nil, fmt.Errorf("settings database is required")
	}
	return &SettingsWriter{db: db}, nil
}

// SetSetting upserts the setting value.
func (w *SettingsWriter) SetSetting(ctx context.Context, key, value string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("setting key is required")
	}
	const sql = `
INSERT INTO app_settings (key, value, updated_at)
VALUES ($1, $2, NOW())
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`
	if _, err := w.db.Exec(ctx, sql, key, value); err != nil {
		return fmt.Errorf("set app setting %q: %w", key, err)
	}
	return nil
}

var _ SettingsWriteStore = (*SettingsWriter)(nil)
