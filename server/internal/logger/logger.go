package logger

import (
	"log/slog"
	"os"
)

// Init initializes the global slog instance with a JSON handler.
// The log level changes based on the environment.
func Init(environment string) {
	level := slog.LevelInfo
	if environment == "development" {
		level = slog.LevelDebug
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)
}
