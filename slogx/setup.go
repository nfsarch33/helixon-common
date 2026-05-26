package slogx

import (
	"log/slog"
	"os"
	"strings"
)

func Setup(component string) *slog.Logger {
	level := slog.LevelInfo
	if l := os.Getenv("LOG_LEVEL"); l != "" {
		switch strings.ToLower(l) {
		case "debug":
			level = slog.LevelDebug
		case "warn", "warning":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		}
	}

	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})

	logger := slog.New(handler).With("component", component)
	slog.SetDefault(logger)
	return logger
}
