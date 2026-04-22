package platform

import "eagleeye/internal/core/timekeeper"

// NewIdleChecker returns a platform-specific idle checker.
func NewIdleChecker() timekeeper.IdleChecker {
	return newIdleProvider()
}
