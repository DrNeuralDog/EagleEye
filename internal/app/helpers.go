package app

import (
	"eagleeye/internal/platform"
	"eagleeye/internal/ui/animation"
	"eagleeye/internal/ui/preferences"
	"eagleeye/resources"
	"fmt"
	"time"
)

func defaultExerciseSpec() animation.ExerciseSpec {
	return animation.ExerciseSpec{
		Instruction: resources.MustSprite("InstractionEagle.png"),
		Center:      resources.MustSprite("Falcon looks straight ahead.png"),
		Left:        resources.MustSprite("Falcon looks left.png"),
		Right:       resources.MustSprite("Falcon looks right.png"),
		Up:          resources.MustSprite("Falcon looks up.png"),
		Down:        resources.MustSprite("Falcon looks down.png"),
		BlinkOpen:   resources.MustSprite("Falcon looks straight ahead.png"),
		BlinkClosed: resources.MustSprite("The falcon squinting is close.png"),
		LookOutside: resources.MustSprite("Picturesque meadow - look outside.png"),
	}
}

func defaultExerciseCycle() []animation.ExerciseType {
	return []animation.ExerciseType{
		animation.ExerciseLeftRight,
		animation.ExerciseUpDown,
		animation.ExerciseBlink,
		animation.ExerciseLookOutside,
	}
}

func formatRemaining(remaining time.Duration) string {
	if remaining < 0 {
		remaining = 0
	}
	seconds := int(remaining.Seconds())
	minutes := seconds / 60
	seconds %= 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func opacityToAlpha(opacity float64) uint8 {
	if opacity < 0 {
		opacity = 0
	}
	if opacity > 1 {
		opacity = 1
	}
	return uint8(opacity * 255)
}

// IsAutostartLaunch reports whether args contain the OS autostart marker.
func IsAutostartLaunch(args []string) bool {
	for _, arg := range args {
		if arg == platform.AutostartArg {
			return true
		}
	}
	return false
}

// ShouldStartTimerOnLaunch reports whether an autostart launch should resume
// the previously started break timer.
func ShouldStartTimerOnLaunch(settings preferences.Settings, autostartLaunch bool) bool {
	return autostartLaunch && settings.RunOnStartup && settings.BreakTimerStarted
}
