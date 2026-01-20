package main

import (
	"fmt"
	"log"
	"time"

	"eagleeye/internal/core/timekeeper"
	"eagleeye/internal/platform"
	"eagleeye/internal/ui/animation"
	"eagleeye/internal/ui/overlay"
	"eagleeye/internal/ui/preferences"
	"eagleeye/internal/ui/tray"
	"eagleeye/resources"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

const appName = "EagleEye"

func main() {
	guard, err := platform.AcquireSingleInstance(appName)
	if err != nil {
		log.Printf("single instance: %v", err)
		return
	}
	defer func() {
		_ = guard.Release()
	}()

	fyneApp := app.NewWithID("com.eagleeye.app")
	fyneApp.SetIcon(resources.MustLogo("Logo_Optimal_Gradient.png"))
	desktopApp, ok := fyneApp.(desktop.App)
	if !ok {
		log.Printf("system tray unsupported on this platform")
		return
	}

	trayWindow := fyneApp.NewWindow("EagleEye")
	trayWindow.SetContent(widget.NewLabel("EagleEye is running in the system tray."))
	trayWindow.SetCloseIntercept(func() {
		trayWindow.Hide()
	})
	trayWindow.Hide()
	desktopApp.SetSystemTrayWindow(trayWindow)

	settings := preferences.DefaultSettings()
	keeper := timekeeper.New(settings.TimeKeeperConfig(), timekeeper.Config{TickInterval: time.Second})
	keeper.SetIdleChecker(platform.NewIdleProvider())

	overlayWindow := overlay.New(fyneApp, overlay.Config{
		Opacity:    opacityToAlpha(settings.OverlayOpacity),
		Fullscreen: settings.Fullscreen,
		Message:    "Time to rest your eyes!",
	}, nil)

	animationEngine := animation.New(animation.DefaultConfig(), func(resource fyne.Resource) {
		overlayWindow.SetSprite(resource)
	})
	overlayWindow.SetEngine(animationEngine)

	overlayWindow.SetOnSkip(func() {
		keeper.SkipBreak()
	})

	exerciseSpec := animation.ExerciseSpec{
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

	idleSpec := animation.IdleSpec{
		Open:   exerciseSpec.Center,
		Closed: exerciseSpec.BlinkClosed,
	}

	isPaused := false
	var pauseTimer *time.Timer
	exerciseIndex := 0
	exerciseCycle := []animation.ExerciseType{
		animation.ExerciseLeftRight,
		animation.ExerciseUpDown,
		animation.ExerciseBlink,
		animation.ExerciseLookOutside,
	}

	started := false
	prefsWindow := preferences.New(fyneApp, settings, func(updated preferences.Settings) {
		settings = updated
		keeper.UpdateConfig(settings.TimeKeeperConfig())
		overlayWindow.UpdateConfig(overlay.Config{
			Opacity:    opacityToAlpha(settings.OverlayOpacity),
			Fullscreen: settings.Fullscreen,
			Message:    "Time to rest your eyes!",
		})
		if !started {
			keeper.Start()
			started = true
		}
	})

	activeIcon := resources.MustLogo("Logo_Bright_Gradient.png")
	pausedIcon := resources.MustLogo("Logo_Dull_Gradient.png")

	var trayManager *tray.Manager
	trayManager = tray.New(desktopApp, tray.Callbacks{
		OnPreferences: func() {
			prefsWindow.Show()
		},
		OnTogglePause: func() {
			if isPaused {
				keeper.Resume()
				isPaused = false
				desktopApp.SetSystemTrayIcon(activeIcon)
			} else {
				keeper.Pause()
				isPaused = true
				desktopApp.SetSystemTrayIcon(pausedIcon)
			}
			trayManager.SetPaused(isPaused)
		},
		OnSkipBreak: func() {
			keeper.SkipBreak()
		},
		OnPauseFor: func(duration time.Duration) {
			if pauseTimer != nil {
				pauseTimer.Stop()
			}
			keeper.Pause()
			isPaused = true
			desktopApp.SetSystemTrayIcon(pausedIcon)
			trayManager.SetPaused(true)
			pauseTimer = time.AfterFunc(duration, func() {
				keeper.Resume()
				isPaused = false
				fyne.Do(func() {
					desktopApp.SetSystemTrayIcon(activeIcon)
					trayManager.SetPaused(false)
				})
			})
		},
		OnForceLong: func() {
			keeper.ForceBreak(timekeeper.StateLongBreak)
		},
		OnQuit: func() {
			keeper.Stop()
			fyneApp.Quit()
		},
	})

	desktopApp.SetSystemTrayIcon(activeIcon)

	events := keeper.Subscribe(5)
	go func() {
		for event := range events {
			switch event.Type {
			case timekeeper.EventStateChange:
				handleStateChange(event, overlayWindow, &exerciseIndex, exerciseCycle, exerciseSpec, idleSpec, trayManager)
			case timekeeper.EventProgress:
				handleProgress(event, overlayWindow, trayManager)
			}
		}
	}()

	prefsWindow.Show()
	fyneApp.Run()
}

func handleStateChange(event timekeeper.Event, overlayWindow *overlay.Window, exerciseIndex *int, cycle []animation.ExerciseType, spec animation.ExerciseSpec, idle animation.IdleSpec, trayManager *tray.Manager) {
	switch event.State {
	case timekeeper.StateShortBreak:
		trayManager.SetInBreak(true)
		exercise := cycle[*exerciseIndex%len(cycle)]
		*exerciseIndex++
		overlayWindow.Show(overlay.Session{
			Remaining:  event.Remaining,
			StrictMode: event.StrictMode,
			Exercise:   exercise,
		}, spec)
	case timekeeper.StateLongBreak:
		trayManager.SetInBreak(true)
		overlayWindow.ShowIdle(event.Remaining, event.StrictMode, idle)
	case timekeeper.StateWork:
		trayManager.SetInBreak(false)
		overlayWindow.Hide()
	case timekeeper.StatePaused:
		trayManager.SetPaused(true)
	}
}

func handleProgress(event timekeeper.Event, overlayWindow *overlay.Window, trayManager *tray.Manager) {
	if event.State == timekeeper.StateShortBreak || event.State == timekeeper.StateLongBreak {
		overlayWindow.SetRemaining(event.Remaining)
	}
	if event.State == timekeeper.StateWork {
		trayManager.SetStatus("next break in " + formatRemaining(event.Remaining))
	}
}

func formatRemaining(remaining time.Duration) string {
	if remaining < 0 {
		remaining = 0
	}
	seconds := int(remaining.Seconds())
	minutes := seconds / 60
	seconds = seconds % 60
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
