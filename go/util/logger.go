package util

import (
	"log/slog"
	"os"
)

func SetupLogger(serviceName string) *slog.Logger {
	var level slog.Leveler
	levelInt, err := GetIntEnv("SLOG_LEVEL", int(slog.LevelDebug))
	if err != nil {
		level = slog.LevelDebug
	} else {
		level = slog.Level(levelInt)
	}
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	logger := slog.New(handler)
	logger = logger.With("service", serviceName)
	slog.SetDefault(logger)

	return logger
}
