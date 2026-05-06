package platform

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"eagleeye/internal/core/timekeeper"
)

type idleProvider struct {
	xprintidlePath string
}

type unsupportedIdleProvider struct{}

// newIdleProvider creates an xprintidle-backed Linux idle checker
func newIdleProvider() timekeeper.IdleChecker {
	path, ok := findAllowedExecutable("xprintidle", linuxExecutableSearchDirs)

	if !ok {
		return unsupportedIdleProvider{}
	}

	return &idleProvider{xprintidlePath: path}
}

// IdleDuration runs xprintidle and converts its millisecond output to a duration
func (provider *idleProvider) IdleDuration() (time.Duration, error) {
	sessionType := strings.ToLower(os.Getenv("XDG_SESSION_TYPE"))

	if sessionType == "wayland" && provider.xprintidlePath == "" {
		return 0, timekeeper.ErrIdleUnsupported
	}

	output, err := exec.Command(provider.xprintidlePath).Output()
	if err != nil {
		return 0, fmt.Errorf("xprintidle: %w", err)
	}

	value := strings.TrimSpace(string(output))
	idleMillis, err := strconv.ParseInt(value, 10, 64)

	if err != nil {
		return 0, fmt.Errorf("parse idle milliseconds: %w", err)
	}

	if idleMillis < 0 {
		idleMillis = 0
	}

	return time.Duration(idleMillis) * time.Millisecond, nil
}

// IdleDuration reports that Linux idle detection is unavailable
func (unsupportedIdleProvider) IdleDuration() (time.Duration, error) {
	return 0, timekeeper.ErrIdleUnsupported
}
