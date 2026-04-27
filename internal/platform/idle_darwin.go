package platform

import (
	"time"

	"eagleeye/internal/core/timekeeper"
)

type idleProvider struct{}

func newIdleProvider() timekeeper.IdleChecker {
	return &idleProvider{}
}

func (provider *idleProvider) IdleDuration() (time.Duration, error) {
	return 0, timekeeper.ErrIdleUnsupported
}
