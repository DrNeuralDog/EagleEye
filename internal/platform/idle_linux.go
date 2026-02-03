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

func newIdleProvider() IdleProvider {
	path, err := exec.LookPath("xprintidle")
	if err != nil {
		return unsupportedIdleProvider{}
	}
	return &idleProvider{xprintidlePath: path}
}

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

func (unsupportedIdleProvider) IdleDuration() (time.Duration, error) {
	return 0, timekeeper.ErrIdleUnsupported
}
