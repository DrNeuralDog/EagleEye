package model

import "time"

// BreakConfig defines a recurring break schedule.
type BreakConfig struct {
	Interval time.Duration
	Duration time.Duration
	Enabled  bool
}

// LongBreakConfig extends BreakConfig with strict mode.
type LongBreakConfig struct {
	BreakConfig
	StrictMode bool
}

// TimeKeeperConfig contains runtime settings for the TimeKeeper state machine.
type TimeKeeperConfig struct {
	Short BreakConfig
	Long  LongBreakConfig

	IdleResetEnabled  bool
	IdleResetAfter    time.Duration
	IdleCheckInterval time.Duration
}
