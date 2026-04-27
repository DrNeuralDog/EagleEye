package timekeeper

import (
	"eagleeye/internal/core/model"
	"testing"
	"time"
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

func TestStopClosesSubscribers(t *testing.T) {
	keeper := newTestKeeper(model.TimeKeeperConfig{})
	events := keeper.Subscribe(2)

	keeper.Start()
	keeper.Stop()

	assertChannelClosed(t, events)
}

func TestStartAfterStopRestartsLoop(t *testing.T) {
	keeper := newTestKeeper(model.TimeKeeperConfig{
		Short: model.BreakConfig{
			Interval: 10 * time.Minute,
			Duration: 15 * time.Second,
			Enabled:  true,
		},
	})

	keeper.Start()
	keeper.Stop()

	events := keeper.Subscribe(4)
	keeper.Start()
	keeper.ForceNextBreak()
	keeper.Stop()

	if state := currentState(keeper); state != StateShortBreak {
		t.Fatalf("state = %s, want %s after restart", state, StateShortBreak)
	}
	assertChannelClosed(t, events)
}

func TestStopWithoutStartAllowsFutureStart(t *testing.T) {
	keeper := newTestKeeper(model.TimeKeeperConfig{})

	keeper.Stop()
	keeper.Start()
	keeper.Stop()
}

func TestStopWithoutStartClosesSubscribers(t *testing.T) {
	keeper := newTestKeeper(model.TimeKeeperConfig{})
	events := keeper.Subscribe(1)

	keeper.Stop()

	assertChannelClosed(t, events)
}

func TestNilIdleCheckerIsNoOp(t *testing.T) {
	keeper := newTestKeeper(model.TimeKeeperConfig{
		IdleResetEnabled:  true,
		IdleResetAfter:    time.Nanosecond,
		IdleCheckInterval: time.Nanosecond,
	})
	keeper.SetIdleChecker(nil)

	keeper.mu.Lock()
	keeper.handleIdleCheckLocked(time.Now())
	idleResetEnabled := keeper.config.IdleResetEnabled
	state := keeper.state
	keeper.mu.Unlock()

	if !idleResetEnabled {
		t.Fatalf("nil idle checker disabled config; want no-op")
	}
	if state != StateWork {
		t.Fatalf("state = %s, want %s", state, StateWork)
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

func assertChannelClosed(t *testing.T, events <-chan Event) {
	t.Helper()

	timeout := time.After(time.Second)
	for {
		select {
		case _, ok := <-events:
			if !ok {
				return
			}
		case <-timeout:
			t.Fatalf("subscriber channel was not closed")
		}
	}
}
