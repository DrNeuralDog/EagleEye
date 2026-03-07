package timekeeper

import (
	"testing"
	"time"

	"eagleeye/internal/core/model"
)

func TestForceNextBreakStartsShortBreakWhenShortIsNext(t *testing.T) {
	keeper := newTestKeeper(model.TimeKeeperConfig{
		Short: model.BreakConfig{
			Interval: 10 * time.Minute,
			Duration: 15 * time.Second,
			Enabled:  true,
		},
		Long: model.LongBreakConfig{
			BreakConfig: model.BreakConfig{
				Interval: 50 * time.Minute,
				Duration: 5 * time.Minute,
				Enabled:  true,
			},
		},
	})
	keeper.Start()
	defer keeper.Stop()

	keeper.ForceNextBreak()

	if state := currentState(keeper); state != StateShortBreak {
		t.Fatalf("state = %s, want %s", state, StateShortBreak)
	}
}

func TestForceNextBreakStartsLongBreakWhenLongIsNext(t *testing.T) {
	keeper := newTestKeeper(model.TimeKeeperConfig{
		Short: model.BreakConfig{
			Interval: 50 * time.Minute,
			Duration: 15 * time.Second,
			Enabled:  true,
		},
		Long: model.LongBreakConfig{
			BreakConfig: model.BreakConfig{
				Interval: 10 * time.Minute,
				Duration: 5 * time.Minute,
				Enabled:  true,
			},
		},
	})
	keeper.Start()
	defer keeper.Stop()

	keeper.ForceNextBreak()

	if state := currentState(keeper); state != StateLongBreak {
		t.Fatalf("state = %s, want %s", state, StateLongBreak)
	}
}

func TestForceNextBreakDoesNothingWhenPaused(t *testing.T) {
	keeper := newTestKeeper(model.TimeKeeperConfig{
		Short: model.BreakConfig{
			Interval: 10 * time.Minute,
			Duration: 15 * time.Second,
			Enabled:  true,
		},
		Long: model.LongBreakConfig{
			BreakConfig: model.BreakConfig{
				Interval: 50 * time.Minute,
				Duration: 5 * time.Minute,
				Enabled:  true,
			},
		},
	})
	keeper.Start()
	defer keeper.Stop()
	keeper.Pause()

	keeper.ForceNextBreak()

	if state := currentState(keeper); state != StatePaused {
		t.Fatalf("state = %s, want %s", state, StatePaused)
	}
}

func newTestKeeper(config model.TimeKeeperConfig) *TimeKeeper {
	return New(config, Config{TickInterval: time.Hour})
}

func currentState(keeper *TimeKeeper) State {
	keeper.mu.Lock()
	defer keeper.mu.Unlock()
	return keeper.state
}
