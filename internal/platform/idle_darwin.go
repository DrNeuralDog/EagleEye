package platform

import (
	"time"

	"eagleeye/internal/core/timekeeper"
)

type idleProvider struct{}

func newIdleProvider() IdleProvider {
	return &idleProvider{}
}

func (provider *idleProvider) IdleDuration() (time.Duration, error) {
	return 0, timekeeper.ErrIdleUnsupported
}
