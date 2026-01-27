package main

import (
	"fmt"
	"log"
	"time"

	"eagleeye/internal/core/timekeeper"
	"eagleeye/internal/platform"
	"eagleeye/internal/storage"
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
	if fyneApp.Icon() != nil {
		trayWindow.SetIcon(fyneApp.Icon())
	}
	trayWindow.SetContent(widget.NewLabel("EagleEye is running in the system tray."))
	trayWindow.SetCloseIntercept(func() {
		trayWindow.Hide()
	})
	trayWindow.Hide()
	desktopApp.SetSystemTrayWindow(trayWindow)

	settings, err := storage.LoadSettings(appName)
	if err != nil {
		log.Printf("load settings: %v", err)
		settings = preferences.DefaultSettings()
	}
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
	serviceStarted := false
	nextBreakRemaining := settings.ShortInterval
	var pauseTimer *time.Timer
	exerciseIndex := 0
	exerciseCycle := []animation.ExerciseType{
		animation.ExerciseLeftRight,
		animation.ExerciseUpDown,
		animation.ExerciseBlink,
		animation.ExerciseLookOutside,
	}

	activeIcon := resources.MustLogo("Logo_Bright_Gradient.png")
	pausedIcon := resources.MustLogo("Logo_Dull_Gradient.png")

	var trayManager *tray.Manager
	var prefsWindow *preferences.Window

	startServiceIfNeeded := func() {
		if serviceStarted {
			return
		}
		keeper.Start()
		serviceStarted = true
		isPaused = false
		desktopApp.SetSystemTrayIcon(activeIcon)
		if trayManager != nil {
			trayManager.SetPaused(false)
		}
		if prefsWindow != nil {
			prefsWindow.SetTimerControlState(true)
			prefsWindow.SetServiceRunning(nextBreakRemaining)
		}
	}

	setPauseState := func(paused bool) {
		if !serviceStarted {
			return
		}

		if paused {
			keeper.Pause()
			isPaused = true
			desktopApp.SetSystemTrayIcon(pausedIcon)
			if trayManager != nil {
				trayManager.SetPaused(true)
			}
			if prefsWindow != nil {
				prefsWindow.SetTimerControlState(false)
				prefsWindow.SetServicePaused()
			}
			return
		}

		keeper.Resume()
		isPaused = false
		desktopApp.SetSystemTrayIcon(activeIcon)
		if trayManager != nil {
			trayManager.SetPaused(false)
		}
		if prefsWindow != nil {
			prefsWindow.SetTimerControlState(true)
			prefsWindow.SetServiceRunning(nextBreakRemaining)
		}
	}

	prefsWindow = preferences.New(fyneApp, settings, preferences.Callbacks{
		OnSave: func(updated preferences.Settings) {
			settings = updated
			if err := storage.SaveSettings(appName, settings); err != nil {
				log.Printf("save settings: %v", err)
			}
			keeper.UpdateConfig(settings.TimeKeeperConfig())
			overlayWindow.UpdateConfig(overlay.Config{
				Opacity:    opacityToAlpha(settings.OverlayOpacity),
				Fullscreen: settings.Fullscreen,
				Message:    "Time to rest your eyes!",
			})
		},
		OnDismiss: func() {
			startServiceIfNeeded()
		},
		OnToggleTimer: func() {
			if isPaused {
				setPauseState(false)
			} else {
				setPauseState(true)
			}
		},
	})
	prefsWindow.SetServiceNotStarted()
	prefsWindow.SetTimerControlState(false)

	trayManager = tray.New(desktopApp, tray.Callbacks{
		OnPreferences: func() {
			prefsWindow.Show()
		},
		OnTogglePause: func() {
			if !serviceStarted {
				return
			}
			if isPaused {
				setPauseState(false)
			} else {
				setPauseState(true)
			}
		},
		OnSkipBreak: func() {
			keeper.SkipBreak()
		},
		OnPauseFor: func(duration time.Duration) {
			if !serviceStarted {
				return
			}
			if pauseTimer != nil {
				pauseTimer.Stop()
			}
			setPauseState(true)
			pauseTimer = time.AfterFunc(duration, func() {
				fyne.Do(func() {
					setPauseState(false)
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
				if event.State == timekeeper.StatePaused {
					nextBreakRemaining = event.Remaining
					isPaused = true
					prefsWindow.SetServicePaused()
					prefsWindow.SetTimerControlState(false)
				}
				if event.State == timekeeper.StateWork && serviceStarted && !isPaused {
					prefsWindow.SetServiceRunning(nextBreakRemaining)
					prefsWindow.SetTimerControlState(true)
				}
			case timekeeper.EventProgress:
				handleProgress(event, overlayWindow, trayManager)
				if event.State == timekeeper.StateWork {
					nextBreakRemaining = event.Remaining
					if serviceStarted && !isPaused {
						prefsWindow.SetServiceRunning(event.Remaining)
						prefsWindow.SetTimerControlState(true)
					}
				}
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
