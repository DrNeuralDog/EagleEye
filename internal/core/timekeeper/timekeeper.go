package timekeeper

import (
	"eagleeye/internal/core/model"
	"errors"
	"sync"
	"time"
)

// ErrIdleUnsupported indicates idle detection is not available on this system.
var ErrIdleUnsupported = errors.New("idle detection unsupported")

// IdleChecker reports the duration of user inactivity.
type IdleChecker interface {
	IdleDuration() (time.Duration, error)
}

// Config contains runtime options for TimeKeeper.
type Config struct {
	TickInterval time.Duration
}

// TimeKeeper is a state machine that manages break scheduling.
//
// Callers own the lifecycle: Start begins the ticker loop, Stop synchronously
// shuts it down, and Subscribe exposes state/progress events. Progress events
// are best-effort and may be dropped when a subscriber is full; non-progress
// events are delivered by discarding stale queued events until space is
// available.
type TimeKeeper struct {
	mu               sync.Mutex
	config           model.TimeKeeperConfig
	options          Config
	state            State
	previousState    State
	remaining        time.Duration
	nextShort        time.Duration
	nextLong         time.Duration
	idleChecker      IdleChecker
	lastIdleCheck    time.Time
	events           []chan Event
	stopCh           chan struct{}
	doneCh           chan struct{}
	running          bool
	paused           bool
	lastProgressSent time.Time
}

// New creates a TimeKeeper with the provided configuration.
func New(config model.TimeKeeperConfig, options Config) *TimeKeeper {
	if options.TickInterval <= 0 {
		options.TickInterval = time.Second
	}
	if config.IdleCheckInterval <= 0 {
		config.IdleCheckInterval = 5 * time.Second
	}

	keeper := &TimeKeeper{
		config:        config,
		options:       options,
		state:         StateWork,
		previousState: StateWork,
	}
	keeper.resetWorkTimersLocked()
	return keeper
}

// SetIdleChecker injects an idle checker. Passing nil disables idle-reset
// checks; timer progression continues without querying idle state.
func (keeper *TimeKeeper) SetIdleChecker(checker IdleChecker) {
	keeper.mu.Lock()
	defer keeper.mu.Unlock()
	keeper.idleChecker = checker
}

// Subscribe registers a new observer channel. Non-positive buffers are
// normalized to 1. Subscriber channels are closed by Stop and are not replayed
// or reused after they have been closed.
func (keeper *TimeKeeper) Subscribe(buffer int) <-chan Event {
	if buffer <= 0 {
		buffer = 1
	}
	ch := make(chan Event, buffer)
	keeper.mu.Lock()
	keeper.events = append(keeper.events, ch)
	keeper.mu.Unlock()
	return ch
}

// Start launches the ticking loop. Calling Start while the keeper is already
// running is a no-op. Calling Start after Stop is supported; it creates a fresh
// stop/done channel pair and emits a StateWork event for current subscribers.
func (keeper *TimeKeeper) Start() {
	keeper.mu.Lock()
	if keeper.running {
		keeper.mu.Unlock()
		return
	}
	stopCh := make(chan struct{})
	doneCh := make(chan struct{})
	keeper.stopCh = stopCh
	keeper.doneCh = doneCh
	keeper.running = true
	keeper.paused = false
	keeper.state = StateWork
	keeper.previousState = StateWork
	keeper.remaining = 0
	keeper.lastIdleCheck = time.Time{}
	keeper.mu.Unlock()

	keeper.emit(Event{
		Type:  EventStateChange,
		State: StateWork,
		At:    time.Now(),
	})

	go keeper.run(stopCh, doneCh)
}

// Stop terminates the ticking loop and closes observers. It waits for the run
// loop to exit before closing subscriber channels. If Start was never called,
// Stop still closes all current subscribers.
func (keeper *TimeKeeper) Stop() {
	keeper.mu.Lock()
	if !keeper.running {
		events := keeper.events
		keeper.events = nil
		keeper.mu.Unlock()
		for _, ch := range events {
			close(ch)
		}
		return
	}
	stopCh := keeper.stopCh
	doneCh := keeper.doneCh
	close(stopCh)
	keeper.running = false
	events := keeper.events
	keeper.events = nil
	keeper.mu.Unlock()

	if doneCh != nil {
		<-doneCh
	}
	for _, ch := range events {
		close(ch)
	}
}

// Pause freezes the timer.
func (keeper *TimeKeeper) Pause() {
	keeper.mu.Lock()
	if keeper.paused {
		keeper.mu.Unlock()
		return
	}
	keeper.paused = true
	remaining := keeper.remaining
	if keeper.state == StateWork {
		remaining = keeper.nextBreakRemainingLocked()
	}
	keeper.previousState = keeper.state
	keeper.state = StatePaused
	keeper.mu.Unlock()

	keeper.emit(Event{
		Type:      EventStateChange,
		State:     StatePaused,
		Remaining: remaining,
		At:        time.Now(),
	})
}

// Resume unfreezes the timer.
func (keeper *TimeKeeper) Resume() {
	keeper.mu.Lock()
	if !keeper.paused {
		keeper.mu.Unlock()
		return
	}
	keeper.paused = false
	keeper.state = keeper.previousState
	currentState := keeper.state
	remaining := keeper.remaining
	if currentState == StateWork {
		remaining = keeper.nextBreakRemainingLocked()
	}
	keeper.mu.Unlock()

	keeper.emit(Event{
		Type:      EventStateChange,
		State:     currentState,
		Remaining: remaining,
		At:        time.Now(),
	})
}

// UpdateConfig updates runtime configuration and resets work timers. If the
// keeper is currently in a break, the active break duration is not recomputed;
// the new work timers take effect when the keeper returns to StateWork.
func (keeper *TimeKeeper) UpdateConfig(config model.TimeKeeperConfig) {
	keeper.mu.Lock()
	if config.IdleCheckInterval <= 0 {
		config.IdleCheckInterval = 5 * time.Second
	}
	keeper.config = config
	keeper.resetWorkTimersLocked()
	keeper.mu.Unlock()
}

// SkipBreak ends the current break and returns to work state.
func (keeper *TimeKeeper) SkipBreak() {
	keeper.mu.Lock()
	if keeper.state != StateShortBreak && keeper.state != StateLongBreak {
		keeper.mu.Unlock()
		return
	}
	keeper.state = StateWork
	keeper.remaining = 0
	keeper.resetWorkTimersLocked()
	keeper.mu.Unlock()

	keeper.emit(Event{
		Type:  EventStateChange,
		State: StateWork,
		At:    time.Now(),
	})
}

// ForceBreak triggers an immediate short or long break.
func (keeper *TimeKeeper) ForceBreak(state State) {
	if state != StateShortBreak && state != StateLongBreak {
		return
	}

	keeper.mu.Lock()
	if !keeper.running || keeper.paused {
		keeper.mu.Unlock()
		return
	}
	keeper.enterBreakLocked(state)
	keeper.mu.Unlock()
}

// ForceNextBreak triggers whichever break would normally happen next.
func (keeper *TimeKeeper) ForceNextBreak() {
	keeper.mu.Lock()
	if !keeper.running || keeper.paused || keeper.state != StateWork {
		keeper.mu.Unlock()
		return
	}
	state, ok := keeper.nextBreakStateLocked()
	if !ok {
		keeper.mu.Unlock()
		return
	}
	keeper.enterBreakLocked(state)
	keeper.mu.Unlock()
}

// ResetForIdle forces the timer to restart work intervals.
func (keeper *TimeKeeper) ResetForIdle() {
	keeper.mu.Lock()
	keeper.resetWorkTimersLocked()
	keeper.mu.Unlock()
}

func (keeper *TimeKeeper) run(stopCh <-chan struct{}, doneCh chan<- struct{}) {
	defer close(doneCh)
	ticker := time.NewTicker(keeper.options.TickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			return
		case tickTime := <-ticker.C:
			keeper.tick(tickTime)
		}
	}
}

func (keeper *TimeKeeper) tick(tickTime time.Time) {
	keeper.mu.Lock()
	if !keeper.running || keeper.paused {
		keeper.mu.Unlock()
		return
	}

	if keeper.state == StateWork {
		keeper.handleIdleCheckLocked(tickTime)
		keeper.advanceWorkLocked(keeper.options.TickInterval)
		keeper.maybeEmitProgressLocked(tickTime)
	} else {
		keeper.advanceBreakLocked(keeper.options.TickInterval, tickTime)
	}
	keeper.mu.Unlock()
}

func (keeper *TimeKeeper) handleIdleCheckLocked(now time.Time) {
	if !keeper.config.IdleResetEnabled || keeper.idleChecker == nil {
		return
	}
	if !keeper.lastIdleCheck.IsZero() && now.Sub(keeper.lastIdleCheck) < keeper.config.IdleCheckInterval {
		return
	}
	keeper.lastIdleCheck = now

	idleDuration, err := keeper.idleChecker.IdleDuration()
	if err != nil {
		if errors.Is(err, ErrIdleUnsupported) {
			keeper.config.IdleResetEnabled = false
			keeper.emitLocked(Event{
				Type:    EventIdleError,
				State:   keeper.state,
				Message: err.Error(),
				At:      now,
			})
			return
		}
		keeper.emitLocked(Event{
			Type:    EventIdleError,
			State:   keeper.state,
			Message: err.Error(),
			At:      now,
		})
		return
	}
	if idleDuration >= keeper.config.IdleResetAfter {
		keeper.resetWorkTimersLocked()
		keeper.emitLocked(Event{
			Type:    EventIdleReset,
			State:   keeper.state,
			Message: "idle reset",
			At:      now,
		})
	}
}

func (keeper *TimeKeeper) advanceWorkLocked(delta time.Duration) {
	if keeper.config.Long.Enabled {
		keeper.nextLong -= delta
		if keeper.nextLong <= 0 {
			keeper.enterBreakLocked(StateLongBreak)
			return
		}
	}
	if keeper.config.Short.Enabled {
		keeper.nextShort -= delta
		if keeper.nextShort <= 0 {
			keeper.enterBreakLocked(StateShortBreak)
			return
		}
	}
}

func (keeper *TimeKeeper) advanceBreakLocked(delta time.Duration, now time.Time) {
	keeper.remaining -= delta
	if keeper.remaining > 0 {
		keeper.emitLocked(Event{
			Type:       EventProgress,
			State:      keeper.state,
			Remaining:  keeper.remaining,
			Progress:   keeper.breakProgressLocked(),
			StrictMode: keeper.config.Long.StrictMode,
			At:         now,
		})
		return
	}

	keeper.state = StateWork
	keeper.remaining = 0
	keeper.resetWorkTimersLocked()

	keeper.emitLocked(Event{
		Type:  EventStateChange,
		State: StateWork,
		At:    now,
	})
}

func (keeper *TimeKeeper) enterBreakLocked(state State) {
	keeper.state = state
	if state == StateLongBreak {
		keeper.remaining = keeper.config.Long.Duration
		keeper.resetWorkTimersLocked()
	} else {
		keeper.remaining = keeper.config.Short.Duration
		keeper.nextShort = keeper.config.Short.Interval
	}

	keeper.emitLocked(Event{
		Type:       EventStateChange,
		State:      state,
		Remaining:  keeper.remaining,
		StrictMode: keeper.config.Long.StrictMode,
		At:         time.Now(),
	})
}

func (keeper *TimeKeeper) resetWorkTimersLocked() {
	keeper.nextShort = keeper.config.Short.Interval
	keeper.nextLong = keeper.config.Long.Interval
}

func (keeper *TimeKeeper) breakProgressLocked() float64 {
	var total time.Duration
	switch keeper.state {
	case StateShortBreak:
		total = keeper.config.Short.Duration
	case StateLongBreak:
		total = keeper.config.Long.Duration
	}
	if total <= 0 {
		return 1
	}
	progress := float64(total-keeper.remaining) / float64(total)
	if progress < 0 {
		return 0
	}
	if progress > 1 {
		return 1
	}
	return progress
}

func (keeper *TimeKeeper) maybeEmitProgressLocked(now time.Time) {
	if keeper.lastProgressSent.IsZero() || now.Sub(keeper.lastProgressSent) >= keeper.options.TickInterval {
		keeper.emitLocked(Event{
			Type:      EventProgress,
			State:     keeper.state,
			Remaining: keeper.nextBreakRemainingLocked(),
			Progress:  keeper.workProgressLocked(),
			At:        now,
		})
		keeper.lastProgressSent = now
	}
}

func (keeper *TimeKeeper) nextBreakRemainingLocked() time.Duration {
	state, ok := keeper.nextBreakStateLocked()
	if !ok {
		return 0
	}
	switch state {
	case StateLongBreak:
		return keeper.nextLong
	case StateShortBreak:
		return keeper.nextShort
	default:
		return 0
	}
}

func (keeper *TimeKeeper) nextBreakStateLocked() (State, bool) {
	if keeper.config.Long.Enabled && (!keeper.config.Short.Enabled || keeper.nextLong < keeper.nextShort) {
		return StateLongBreak, true
	}
	if keeper.config.Short.Enabled {
		return StateShortBreak, true
	}
	return "", false
}

func (keeper *TimeKeeper) workProgressLocked() float64 {
	if keeper.config.Long.Enabled && keeper.config.Long.Interval > 0 {
		return float64(keeper.config.Long.Interval-keeper.nextLong) / float64(keeper.config.Long.Interval)
	}
	if keeper.config.Short.Enabled && keeper.config.Short.Interval > 0 {
		return float64(keeper.config.Short.Interval-keeper.nextShort) / float64(keeper.config.Short.Interval)
	}
	return 0
}

func (keeper *TimeKeeper) emit(event Event) {
	keeper.mu.Lock()
	defer keeper.mu.Unlock()
	keeper.emitLocked(event)
}

func (keeper *TimeKeeper) emitLocked(event Event) {
	events := append([]chan Event(nil), keeper.events...)
	for _, ch := range events {
		keeper.enqueueEventLocked(ch, event)
	}
}

func (keeper *TimeKeeper) enqueueEventLocked(ch chan Event, event Event) {
	// Progress events are best-effort; state changes must be delivered.
	if event.Type == EventProgress {
		select {
		case ch <- event:
		default:
		}
		return
	}

	for {
		select {
		case ch <- event:
			return
		default:
			// Drop one queued event (typically stale progress) and retry.
			select {
			case <-ch:
			default:
			}
		}
	}
}
