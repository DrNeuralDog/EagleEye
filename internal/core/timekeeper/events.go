package timekeeper

import "time"

// State represents the current TimeKeeper mode.
type State string

const (
	StateWork       State = "work"
	StateShortBreak State = "short_break"
	StateLongBreak  State = "long_break"
	StatePaused     State = "paused"
)

// EventType defines the type of TimeKeeper event.
type EventType string

const (
	EventStateChange EventType = "state_change"
	EventProgress    EventType = "progress"
	EventIdleReset   EventType = "idle_reset"
	EventIdleError   EventType = "idle_error"
)

// Event represents a TimeKeeper update for observers.
type Event struct {
	Type       EventType
	State      State
	Remaining  time.Duration
	Progress   float64
	StrictMode bool
	Message    string
	At         time.Time
}
