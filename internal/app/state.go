package app

import (
	"eagleeye/internal/ui/animation"
	"sync"
	"time"
)

// appState stores UI-facing runtime state guarded by a mutex
type appState struct {
	mu sync.Mutex

	serviceStarted     bool
	paused             bool
	nextBreakRemaining time.Duration
	pauseTimer         *time.Timer
	exerciseIndex      int
}

// newAppState creates state with the initial work countdown
func newAppState(initialNextBreak time.Duration) *appState {
	return &appState{nextBreakRemaining: initialNextBreak}
}

// ServiceStarted reports whether the break timer has been started
func (state *appState) ServiceStarted() bool {
	state.mu.Lock()
	defer state.mu.Unlock()

	return state.serviceStarted
}

// IsPaused reports whether timer progression is currently paused
func (state *appState) IsPaused() bool {
	state.mu.Lock()
	defer state.mu.Unlock()

	return state.paused
}

// Start marks the service as running and clears pause state
func (state *appState) Start() {
	state.mu.Lock()
	defer state.mu.Unlock()

	state.serviceStarted = true
	state.paused = false
}

// SetPaused updates the cached pause state
func (state *appState) SetPaused(paused bool) {
	state.mu.Lock()
	defer state.mu.Unlock()

	state.paused = paused
}

// SetNextBreakRemaining stores the latest countdown shown by UI surfaces
func (state *appState) SetNextBreakRemaining(remaining time.Duration) {
	state.mu.Lock()
	defer state.mu.Unlock()

	state.nextBreakRemaining = remaining
}

// NextBreakRemaining returns the latest cached work countdown
func (state *appState) NextBreakRemaining() time.Duration {
	state.mu.Lock()
	defer state.mu.Unlock()

	return state.nextBreakRemaining
}

// NextExercise advances through the configured exercise cycle
func (state *appState) NextExercise(cycle []animation.ExerciseType) animation.ExerciseType {
	state.mu.Lock()
	defer state.mu.Unlock()

	if len(cycle) == 0 {
		return animation.ExerciseLeftRight
	}

	exercise := cycle[state.exerciseIndex%len(cycle)]
	state.exerciseIndex++

	return exercise
}

// StopPauseTimer cancels and clears the active auto-resume timer
func (state *appState) StopPauseTimer() {
	timer := state.takePauseTimer()

	if timer != nil {
		timer.Stop()
	}
}

// SetPauseTimer replaces the active auto-resume timer
func (state *appState) SetPauseTimer(timer *time.Timer) {
	state.mu.Lock()
	previous := state.pauseTimer
	state.pauseTimer = timer
	state.mu.Unlock()

	if previous != nil {
		previous.Stop()
	}
}

// takePauseTimer clears and returns the active auto-resume timer
func (state *appState) takePauseTimer() *time.Timer {
	state.mu.Lock()
	defer state.mu.Unlock()

	timer := state.pauseTimer
	state.pauseTimer = nil

	return timer
}
