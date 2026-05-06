package app

import (
	"eagleeye/internal/platform"
	"eagleeye/internal/ui/animation"
	"eagleeye/internal/ui/preferences"
	"testing"
	"time"
)

// TestIsAutostartLaunch verifies detection of the platform autostart marker
func TestIsAutostartLaunch(t *testing.T) {
	if !IsAutostartLaunch([]string{platform.AutostartArg}) {
		t.Fatalf("IsAutostartLaunch() = false, want true")
	}

	if IsAutostartLaunch([]string{"--other"}) {
		t.Fatalf("IsAutostartLaunch() = true, want false")
	}
}

// TestShouldStartTimerOnLaunch locks the autostart resume conditions
func TestShouldStartTimerOnLaunch(t *testing.T) {
	settings := preferences.DefaultSettings()
	settings.RunOnStartup = true
	settings.BreakTimerStarted = true

	if !ShouldStartTimerOnLaunch(settings, true) {
		t.Fatalf("ShouldStartTimerOnLaunch() = false, want true")
	}

	settings.BreakTimerStarted = false
	if ShouldStartTimerOnLaunch(settings, true) {
		t.Fatalf("ShouldStartTimerOnLaunch() with never-started timer = true, want false")
	}

	settings.BreakTimerStarted = true
	if ShouldStartTimerOnLaunch(settings, false) {
		t.Fatalf("ShouldStartTimerOnLaunch() on manual launch = true, want false")
	}
}

// TestAppStateExerciseCycle verifies exercise rotation and empty-cycle fallback
func TestAppStateExerciseCycle(t *testing.T) {
	state := newAppState(time.Minute)
	cycle := []animation.ExerciseType{animation.ExerciseBlink, animation.ExerciseLookOutside}

	if got := state.NextExercise(cycle); got != animation.ExerciseBlink {
		t.Fatalf("first exercise = %v, want blink", got)
	}

	if got := state.NextExercise(cycle); got != animation.ExerciseLookOutside {
		t.Fatalf("second exercise = %v, want look outside", got)
	}

	if got := state.NextExercise(cycle); got != animation.ExerciseBlink {
		t.Fatalf("third exercise = %v, want cycle restart", got)
	}

	if got := state.NextExercise(nil); got != animation.ExerciseLeftRight {
		t.Fatalf("empty cycle exercise = %v, want left/right", got)
	}
}

// TestAppStatePauseTimerReplacementAndStop verifies timer cleanup semantics
func TestAppStatePauseTimerReplacementAndStop(t *testing.T) {
	state := newAppState(time.Minute)
	first := time.NewTimer(time.Hour)
	second := time.NewTimer(time.Hour)

	defer second.Stop()

	state.SetPauseTimer(first)
	state.SetPauseTimer(second)

	if first.Stop() {
		t.Fatalf("first timer was not stopped when replaced")
	}

	if got := state.takePauseTimer(); got != second {
		t.Fatalf("stored timer = %p, want second timer %p", got, second)
	}

	state.SetPauseTimer(time.NewTimer(time.Hour))
	state.StopPauseTimer()

	if got := state.takePauseTimer(); got != nil {
		t.Fatalf("pause timer = %p, want nil after StopPauseTimer", got)
	}
}

// TestFormatRemaining verifies countdown formatting and negative clamping
func TestFormatRemaining(t *testing.T) {
	tests := []struct {
		name      string
		remaining time.Duration
		want      string
	}{
		{name: "negative", remaining: -time.Second, want: "00:00"},
		{name: "zero", remaining: 0, want: "00:00"},
		{name: "minutes and seconds", remaining: 65 * time.Second, want: "01:05"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatRemaining(tt.remaining); got != tt.want {
				t.Fatalf("formatRemaining() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestOpacityToAlpha verifies opacity clamping and alpha conversion
func TestOpacityToAlpha(t *testing.T) {
	tests := []struct {
		name    string
		opacity float64
		want    uint8
	}{
		{name: "below zero", opacity: -1, want: 0},
		{name: "half", opacity: 0.5, want: 127},
		{name: "above one", opacity: 2, want: 255},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := opacityToAlpha(tt.opacity); got != tt.want {
				t.Fatalf("opacityToAlpha() = %d, want %d", got, tt.want)
			}
		})
	}
}
