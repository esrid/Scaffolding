package config

import (
	"log/slog"
	"os"
)

func NewSlog(env string) *slog.Logger {
	var handler slog.Handler
	var level slog.Level
	if env == "production" {
		level = slog.LevelInfo
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	} else {
		level = slog.LevelDebug
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	}
	return slog.New(handler)
}
