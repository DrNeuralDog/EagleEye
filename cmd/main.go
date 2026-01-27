package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
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

type jsonLogger struct {
	mu   sync.Mutex
	file *os.File
	enc  *json.Encoder
}

func newJSONLogger(filename string) *jsonLogger {
	if filename == "" {
		return nil
	}
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		log.Printf("open log file: %v", err)
		return nil
	}
	return &jsonLogger{
		file: file,
		enc:  json.NewEncoder(file),
	}
}

func (logger *jsonLogger) Close() {
	if logger == nil || logger.file == nil {
		return
	}
	_ = logger.file.Close()
}

func (logger *jsonLogger) Log(event string, fields map[string]any) {
	if logger == nil {
		return
	}
	payload := map[string]any{
		"ts":    time.Now().Format(time.RFC3339Nano),
		"event": event,
	}
	for key, value := range fields {
		payload[key] = value
	}
	logger.mu.Lock()
	defer logger.mu.Unlock()
	if err := logger.enc.Encode(payload); err != nil {
		log.Printf("write log: %v", err)
	}
}

func main() {
	guard, err := platform.AcquireSingleInstance(appName)
	if err != nil {
		if errors.Is(err, platform.ErrAlreadyRunning) {
			if notifyErr := platform.NotifyRunningInstance(appName); notifyErr != nil {
				log.Printf("notify running instance: %v", notifyErr)
			}
		}
		log.Printf("single instance: %v", err)
		return
	}
	defer func() {
		_ = guard.Release()
	}()

	exePath, err := os.Executable()
	if err != nil {
		log.Printf("resolve executable: %v", err)
	}
	logPath := ""
	if exePath != "" {
		logPath = filepath.Join(filepath.Dir(exePath), "EagleEye.log.jsonl")
	}
	jsonLog := newJSONLogger(logPath)
	defer jsonLog.Close()

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
	animationEngine.SetOnExerciseChange(func(exercise animation.ExerciseType) {
		overlayWindow.SetExercise(exercise)
	})
	overlayWindow.SetEngine(animationEngine)

	overlayWindow.SetOnSkip(func() {
		jsonLog.Log("break_skip", map[string]any{
			"state": "skip",
		})
		overlayWindow.Hide()
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
	guard.ListenForActivation(func() {
		fyne.Do(func() {
			prefsWindow.Show()
		})
	})

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
	lastState := timekeeper.State("")
	go func() {
		for event := range events {
			switch event.Type {
			case timekeeper.EventStateChange:
				prevState := lastState
				jsonLog.Log("state_change", map[string]any{
					"from":      prevState,
					"to":        event.State,
					"remaining": event.Remaining.String(),
					"strict":    event.StrictMode,
				})
				if event.State == timekeeper.StateShortBreak || event.State == timekeeper.StateLongBreak {
					jsonLog.Log("break_start", map[string]any{
						"type":      event.State,
						"remaining": event.Remaining.String(),
						"strict":    event.StrictMode,
					})
				}
				if event.State == timekeeper.StateWork && (prevState == timekeeper.StateShortBreak || prevState == timekeeper.StateLongBreak) {
					jsonLog.Log("break_complete", map[string]any{
						"from": prevState,
					})
				}
				lastState = event.State
				handleStateChange(event, overlayWindow, &exerciseIndex, exerciseCycle, exerciseSpec, idleSpec, trayManager, jsonLog)
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
				handleProgress(event, overlayWindow, trayManager, jsonLog)
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

func handleStateChange(event timekeeper.Event, overlayWindow *overlay.Window, exerciseIndex *int, cycle []animation.ExerciseType, spec animation.ExerciseSpec, idle animation.IdleSpec, trayManager *tray.Manager, logger *jsonLogger) {
	switch event.State {
	case timekeeper.StateShortBreak:
		trayManager.SetInBreak(true)
		exercise := cycle[*exerciseIndex%len(cycle)]
		*exerciseIndex++
		if logger != nil {
			logger.Log("overlay_show_called", map[string]any{
				"type":      "short_break",
				"remaining": event.Remaining.String(),
				"strict":    event.StrictMode,
			})
		}
		fyne.Do(func() {
			if logger != nil {
				logger.Log("overlay_show_done", map[string]any{
					"type":      "short_break",
					"remaining": event.Remaining.String(),
					"strict":    event.StrictMode,
				})
			}
			overlayWindow.Show(overlay.Session{
				Remaining:  event.Remaining,
				StrictMode: event.StrictMode,
				Exercise:   exercise,
			}, spec)
		})
	case timekeeper.StateLongBreak:
		trayManager.SetInBreak(true)
		if logger != nil {
			logger.Log("overlay_show_called", map[string]any{
				"type":      "long_break",
				"remaining": event.Remaining.String(),
				"strict":    event.StrictMode,
			})
		}
		fyne.Do(func() {
			if logger != nil {
				logger.Log("overlay_show_done", map[string]any{
					"type":      "long_break",
					"remaining": event.Remaining.String(),
					"strict":    event.StrictMode,
				})
			}
			overlayWindow.ShowIdle(event.Remaining, event.StrictMode, idle)
		})
	case timekeeper.StateWork:
		trayManager.SetInBreak(false)
		if logger != nil {
			logger.Log("overlay_hide_called", map[string]any{
				"reason": "state_work",
			})
		}
		fyne.Do(func() {
			overlayWindow.Hide()
			if logger != nil {
				logger.Log("overlay_hide_done", map[string]any{
					"reason": "state_work",
				})
			}
		})
	case timekeeper.StatePaused:
		trayManager.SetPaused(true)
	}
}

func handleProgress(event timekeeper.Event, overlayWindow *overlay.Window, trayManager *tray.Manager, logger *jsonLogger) {
	if event.State == timekeeper.StateShortBreak || event.State == timekeeper.StateLongBreak {
		if event.Remaining <= 0 && logger != nil {
			logger.Log("overlay_hide_called", map[string]any{
				"reason": "progress_done",
			})
		}
		fyne.Do(func() {
			overlayWindow.SetRemaining(event.Remaining)
			if event.Remaining <= 0 {
				trayManager.SetInBreak(false)
				overlayWindow.Hide()
				if logger != nil {
					logger.Log("overlay_hide_done", map[string]any{
						"reason": "progress_done",
					})
				}
			}
		})
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
