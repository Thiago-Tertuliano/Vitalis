package logger

import (
	"log/slog"
	"os"
)

func New(service, env string) *slog.Logger {
	level := slog.LevelInfo
	if env == "local" || env == "dev" {
		level = slog.LevelDebug
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return slog.New(h).With("service", service, "env", env)
}

