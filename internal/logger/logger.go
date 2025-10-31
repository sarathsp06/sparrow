package logger

import (
	"log/slog"
	"os"

	slogotel "github.com/remychantenay/slog-otel"
)

// Logger provides structured logging using slog
var Logger *slog.Logger

func init() {
	SetLevel(slog.LevelInfo)
}

// NewLogger creates a new logger with the given name
func NewLogger(name string) *slog.Logger {
	return Logger.With("component", name)
}

// SetLevel sets the logging level
func SetLevel(level slog.Level) {
	slog.SetDefault(slog.New(slogotel.OtelHandler{
		Next: slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:     level,
			AddSource: true,
		}),
	}))
	Logger = slog.Default()
}
