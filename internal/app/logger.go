package app

import (
	"eagleeye/internal/platform"
	"errors"
	"log/slog"
	"net"
)

func releaseGuard(logger *slog.Logger, guard *platform.InstanceGuard) {
	if err := guard.Release(); err != nil && !errors.Is(err, net.ErrClosed) {
		logger.Warn("release single instance guard", "error", err)
	}
}
