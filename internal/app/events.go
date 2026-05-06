package app

import (
	"eagleeye/internal/core/timekeeper"
	"eagleeye/internal/ui/overlay"
	"sync"

	"fyne.io/fyne/v2"
)

// consumeEvents routes TimeKeeper events to UI handlers until the channel closes
func (rt *AppController) consumeEvents(wg *sync.WaitGroup, events <-chan timekeeper.Event) {
	defer wg.Done()

	lastState := timekeeper.State("")

	for event := range events {
		switch event.Type {
		case timekeeper.EventStateChange:
			previousState := lastState
			rt.logStateChange(previousState, event)
			lastState = event.State
			rt.handleStateChange(event)
		case timekeeper.EventProgress:
			rt.handleProgress(event)
		}
	}
}

// logStateChange records state transitions and break lifecycle markers
func (rt *AppController) logStateChange(previousState timekeeper.State, event timekeeper.Event) {
	rt.logger.Info("state_change",
		"from", string(previousState),
		"to", string(event.State),
		"remaining", event.Remaining.String(),
		"strict", event.StrictMode,
	)

	if event.State == timekeeper.StateShortBreak || event.State == timekeeper.StateLongBreak {
		rt.logger.Info("break_start",
			"type", string(event.State),
			"remaining", event.Remaining.String(),
			"strict", event.StrictMode,
		)
	}

	if event.State == timekeeper.StateWork && (previousState == timekeeper.StateShortBreak || previousState == timekeeper.StateLongBreak) {
		rt.logger.Info("break_complete", "from", string(previousState))
	}
}

// handleStateChange dispatches state transitions to concrete UI reactions
func (rt *AppController) handleStateChange(event timekeeper.Event) {
	switch event.State {
	case timekeeper.StateShortBreak:
		rt.handleShortBreak(event)
	case timekeeper.StateLongBreak:
		rt.handleLongBreak(event)
	case timekeeper.StateWork:
		rt.handleWorkState()
	case timekeeper.StatePaused:
		rt.handlePausedState(event)
	}
}

// handleShortBreak starts an exercise overlay for a short break
func (rt *AppController) handleShortBreak(event timekeeper.Event) {
	rt.trayManager.SetInBreak(true)
	exercise := rt.state.NextExercise(rt.exerciseCycle)

	rt.logger.Info("overlay_show_called",
		"type", "short_break",
		"remaining", event.Remaining.String(),
		"strict", event.StrictMode,
	)
	fyne.Do(func() {
		rt.logger.Info("overlay_show_done",
			"type", "short_break",
			"remaining", event.Remaining.String(),
			"strict", event.StrictMode,
		)
		rt.overlayWindow.Show(overlay.Session{
			Remaining:  event.Remaining,
			StrictMode: event.StrictMode,
			Exercise:   exercise,
		}, rt.exerciseSpec)
	})
}

// handleLongBreak starts the idle overlay for a long break
func (rt *AppController) handleLongBreak(event timekeeper.Event) {
	rt.trayManager.SetInBreak(true)

	rt.logger.Info("overlay_show_called",
		"type", "long_break",
		"remaining", event.Remaining.String(),
		"strict", event.StrictMode,
	)

	fyne.Do(func() {
		rt.logger.Info("overlay_show_done",
			"type", "long_break",
			"remaining", event.Remaining.String(),
			"strict", event.StrictMode,
		)

		rt.overlayWindow.ShowIdle(event.Remaining, event.StrictMode, rt.idleSpec)
	})
}

func (rt *AppController) handleWorkState() {
	rt.trayManager.SetInBreak(false)
	rt.logger.Info("overlay_hide_called", "reason", "state_work")

	fyne.Do(func() {
		rt.overlayWindow.Hide()
		rt.logger.Info("overlay_hide_done", "reason", "state_work")
	})

	if rt.state.ServiceStarted() && !rt.state.IsPaused() {
		rt.prefsWindow.SetServiceRunning(rt.state.NextBreakRemaining())
		rt.prefsWindow.SetTimerControlState(true)
	}
}

// handlePausedState mirrors pause state into tray and preferences UI
func (rt *AppController) handlePausedState(event timekeeper.Event) {
	rt.trayManager.SetPaused(true)
	rt.state.SetNextBreakRemaining(event.Remaining)
	rt.state.SetPaused(true)
	rt.prefsWindow.SetServicePaused()
	rt.prefsWindow.SetTimerControlState(false)
}

// handleProgress updates the active work or break UI countdown
func (rt *AppController) handleProgress(event timekeeper.Event) {
	if event.State == timekeeper.StateShortBreak || event.State == timekeeper.StateLongBreak {
		rt.handleBreakProgress(event)
	}

	if event.State == timekeeper.StateWork {
		rt.handleWorkProgress(event)
	}
}

func (rt *AppController) handleBreakProgress(event timekeeper.Event) {
	if event.Remaining <= 0 {
		rt.logger.Info("overlay_hide_called", "reason", "progress_done")
	}

	fyne.Do(func() {
		rt.overlayWindow.SetRemaining(event.Remaining)

		if event.Remaining <= 0 {
			rt.trayManager.SetInBreak(false)
			rt.overlayWindow.Hide()
			rt.logger.Info("overlay_hide_done", "reason", "progress_done")
		}
	})
}

// handleWorkProgress updates cached countdown, tray text, and preferences status
func (rt *AppController) handleWorkProgress(event timekeeper.Event) {
	rt.state.SetNextBreakRemaining(event.Remaining)
	rt.trayManager.SetStatus(rt.localizer.T("tray.nextBreakIn", formatRemaining(event.Remaining)))

	if rt.state.ServiceStarted() && !rt.state.IsPaused() {
		rt.prefsWindow.SetServiceRunning(event.Remaining)
		rt.prefsWindow.SetTimerControlState(true)
	}
}
