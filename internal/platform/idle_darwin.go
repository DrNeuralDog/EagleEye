package platform

import (
	"time"

	"eagleeye/internal/core/timekeeper"
)

type idleProvider struct{}

// newIdleProvider creates the macOS idle checker placeholder
func newIdleProvider() timekeeper.IdleChecker {
	return &idleProvider{}
}

// IdleDuration reports that macOS idle detection is unavailable
func (provider *idleProvider) IdleDuration() (time.Duration, error) {
	return 0, timekeeper.ErrIdleUnsupported
}
