package log

import (
	"io"
	"log/slog"
)

// New creates a JSON logger with the module field required by the platform contract.
func New(writer io.Writer, level slog.Level, module string) *slog.Logger {
	handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: level}).WithAttrs([]slog.Attr{
		slog.String("module", module),
	})
	return slog.New(handler)
}
