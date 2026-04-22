package app

import (
	"eagleeye/internal/ui/animation"
	"sync"
	"time"
)

type appState struct {
	mu sync.Mutex

	serviceStarted     bool
	paused             bool
	nextBreakRemaining time.Duration
	pauseTimer         *time.Timer
	exerciseIndex      int
}

func newAppState(initialNextBreak time.Duration) *appState {
	return &appState{nextBreakRemaining: initialNextBreak}
}

func (state *appState) ServiceStarted() bool {
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.serviceStarted
}

func (state *appState) IsPaused() bool {
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.paused
}

func (state *appState) Start() {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.serviceStarted = true
	state.paused = false
}

func (state *appState) SetPaused(paused bool) {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.paused = paused
}

func (state *appState) SetNextBreakRemaining(remaining time.Duration) {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.nextBreakRemaining = remaining
}

func (state *appState) NextBreakRemaining() time.Duration {
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.nextBreakRemaining
}

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

func (state *appState) StopPauseTimer() {
	timer := state.takePauseTimer()
	if timer != nil {
		timer.Stop()
	}
}

func (state *appState) SetPauseTimer(timer *time.Timer) {
	state.mu.Lock()
	previous := state.pauseTimer
	state.pauseTimer = timer
	state.mu.Unlock()

	if previous != nil {
		previous.Stop()
	}
}

func (state *appState) takePauseTimer() *time.Timer {
	state.mu.Lock()
	defer state.mu.Unlock()
	timer := state.pauseTimer
	state.pauseTimer = nil
	return timer
}
