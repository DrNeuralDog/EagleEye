package platform

import "time"

// IdleProvider returns the duration since last user input.
type IdleProvider interface {
	IdleDuration() (time.Duration, error)
}

// NewIdleProvider returns a platform-specific idle provider.
func NewIdleProvider() IdleProvider {
	return newIdleProvider()
}
