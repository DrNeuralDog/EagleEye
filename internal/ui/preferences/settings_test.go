package preferences

import (
	"testing"
	"time"
)

// TestSettingsTimeKeeperConfigUsesTwentySecondIdleCheck locks the Preferences
// to TimeKeeper mapping so the idle detector keeps its intended polling cadence!
func TestSettingsTimeKeeperConfigUsesTwentySecondIdleCheck(t *testing.T) {
	config := DefaultSettings().TimeKeeperConfig()

	if config.IdleCheckInterval != 20*time.Second {
		t.Fatalf("IdleCheckInterval = %s, want 20s", config.IdleCheckInterval)
	}
}
