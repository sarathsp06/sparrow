package logger

import (
	"log/slog"
	"os"
)

// Logger provides structured logging using slog
var Logger *slog.Logger

func init() {
	// Create a structured logger with JSON output
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	// Use JSON handler for structured logging
	handler := slog.NewJSONHandler(os.Stdout, opts)
	Logger = slog.New(handler)
}

// NewLogger creates a new logger with the given name
func NewLogger(name string) *slog.Logger {
	return Logger.With("component", name)
}

// SetLevel sets the logging level
func SetLevel(level slog.Level) {
	opts := &slog.HandlerOptions{
		Level: level,
	}
	handler := slog.NewJSONHandler(os.Stdout, opts)
	Logger = slog.New(handler)
}
