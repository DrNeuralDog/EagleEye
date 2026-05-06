package logging

import (
	"eagleeye/internal/storage"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// NewJSONLogger creates the process logger backed by a JSON slog handler
func NewJSONLogger(appName string) (*slog.Logger, func()) {
	level := new(slog.LevelVar)
	level.Set(LevelFromEnv())

	handlerOptions := &slog.HandlerOptions{Level: level}
	fallback := slog.New(slog.NewJSONHandler(os.Stderr, handlerOptions))

	logPath, err := storage.ResolveLogPath(appName)

	if err != nil {
		fallback.Warn("resolve log path", "error", err)

		return fallback, func() {}
	}

	if err := os.MkdirAll(filepath.Dir(logPath), 0o700); err != nil {
		fallback.Warn("create log directory", "error", err)

		return fallback, func() {}
	}

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		fallback.Warn("open log file", "error", err)

		return fallback, func() {}
	}

	if err := file.Chmod(0o600); err != nil {
		fallback.Warn("secure log file permissions", "error", err)

		if closeErr := file.Close(); closeErr != nil {
			fallback.Warn("close log file after permission failure", "error", closeErr)
		}

		return fallback, func() {}
	}

	logger := slog.New(slog.NewJSONHandler(file, handlerOptions))

	return logger, func() {
		if err := file.Close(); err != nil {
			fallback.Warn("close log file", "error", err)
		}
	}
}

// LevelFromEnv parses EAGLEEYE_LOG_LEVEL
func LevelFromEnv() slog.Level {
	switch strings.ToLower(os.Getenv("EAGLEEYE_LOG_LEVEL")) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
