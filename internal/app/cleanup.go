package app

import (
	"eagleeye/internal/platform"
	"errors"
	"log/slog"
	"net"
)

// releaseGuard releases the single-instance lock and logs real cleanup errors
func releaseGuard(logger *slog.Logger, guard *platform.InstanceGuard) {
	if err := guard.Release(); err != nil && !errors.Is(err, net.ErrClosed) {
		logger.Warn("release single instance guard", "error", err)
	}
}
