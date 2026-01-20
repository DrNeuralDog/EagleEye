package animation

import "time"

import "fyne.io/fyne/v2"

// ExerciseSpec defines the sprite set for a single exercise session.
type ExerciseSpec struct {
	Type        ExerciseType
	Duration    time.Duration
	Instruction fyne.Resource
	Center      fyne.Resource
	Left        fyne.Resource
	Right       fyne.Resource
	Up          fyne.Resource
	Down        fyne.Resource
	BlinkOpen   fyne.Resource
	BlinkClosed fyne.Resource
	LookOutside fyne.Resource
}

// BlinkHoldDuration returns the blink hold duration based on the iteration.
func (spec ExerciseSpec) BlinkHoldDuration(longHold bool) time.Duration {
	if longHold {
		return 3 * time.Second
	}
	return time.Second
}

// IdleSpec defines sprites used for idle blinking.
type IdleSpec struct {
	Open   fyne.Resource
	Closed fyne.Resource
}
