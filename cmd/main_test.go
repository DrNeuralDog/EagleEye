package main

import (
	"testing"

	"eagleeye/internal/platform"
	"eagleeye/internal/ui/preferences"
)

func TestIsAutostartLaunch(t *testing.T) {
	if !isAutostartLaunch([]string{platform.AutostartArg}) {
		t.Fatalf("isAutostartLaunch() = false, want true")
	}
	if isAutostartLaunch([]string{"--other"}) {
		t.Fatalf("isAutostartLaunch() = true, want false")
	}
}

func TestShouldStartTimerOnLaunch(t *testing.T) {
	settings := preferences.DefaultSettings()
	settings.RunOnStartup = true
	settings.BreakTimerStarted = true

	if !shouldStartTimerOnLaunch(settings, true) {
		t.Fatalf("shouldStartTimerOnLaunch() = false, want true")
	}

	settings.BreakTimerStarted = false
	if shouldStartTimerOnLaunch(settings, true) {
		t.Fatalf("shouldStartTimerOnLaunch() with never-started timer = true, want false")
	}

	settings.BreakTimerStarted = true
	if shouldStartTimerOnLaunch(settings, false) {
		t.Fatalf("shouldStartTimerOnLaunch() on manual launch = true, want false")
	}
}
