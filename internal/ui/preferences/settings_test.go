package preferences

import (
	"testing"
	"time"
)

func TestSettingsTimeKeeperConfigUsesTwentySecondIdleCheck(t *testing.T) {
	config := DefaultSettings().TimeKeeperConfig()

	if config.IdleCheckInterval != 20*time.Second {
		t.Fatalf("IdleCheckInterval = %s, want 20s", config.IdleCheckInterval)
	}
}
