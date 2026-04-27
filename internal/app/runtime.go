package app

import (
	"context"
	"eagleeye/internal/core/timekeeper"
	"eagleeye/internal/logging"
	"eagleeye/internal/platform"
	"eagleeye/internal/storage"
	"eagleeye/internal/ui/animation"
	"eagleeye/internal/ui/i18n"
	"eagleeye/internal/ui/overlay"
	"eagleeye/internal/ui/preferences"
	"eagleeye/internal/ui/tray"
	"eagleeye/resources"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

const appName = "EagleEye"

// Run starts the EagleEye desktop application and blocks until the UI exits.
func Run(ctx context.Context, args []string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	logger, closeLogger := logging.NewJSONLogger(appName)
	defer closeLogger()

	autostartLaunch := IsAutostartLaunch(args)
	guard, err := platform.AcquireSingleInstance(appName)
	if err != nil {
		if errors.Is(err, platform.ErrAlreadyRunning) {
			if !autostartLaunch {
				if notifyErr := platform.NotifyRunningInstance(appName); notifyErr != nil {
					logger.Warn("notify running instance", "error", notifyErr)
				}
			}
			logger.Info("single instance already running")
			return nil
		}
		return fmt.Errorf("acquire single instance: %w", err)
	}
	defer releaseGuard(logger, guard)

	rt, err := newRuntime(ctx, logger)
	if err != nil {
		return err
	}
	defer rt.keeper.Stop()

	var eventWG sync.WaitGroup
	events := rt.keeper.Subscribe(5)
	eventWG.Add(1)
	go rt.consumeEvents(&eventWG, events)
	defer eventWG.Wait()

	guard.ListenForActivation(ctx, func() {
		fyne.Do(func() {
			rt.prefsWindow.Show()
		})
	})

	go func() {
		<-ctx.Done()
		rt.keeper.Stop()
		fyne.Do(func() {
			rt.fyneApp.Quit()
		})
	}()

	if ShouldStartTimerOnLaunch(rt.settings, autostartLaunch) {
		rt.startServiceIfNeeded()
	} else {
		rt.prefsWindow.Show()
	}

	rt.fyneApp.Run()
	rt.keeper.Stop()
	eventWG.Wait()
	return nil
}

func newRuntime(ctx context.Context, logger *slog.Logger) (*AppController, error) {
	exePath, err := os.Executable()
	if err != nil {
		logger.Warn("resolve executable", "error", err)
	}

	platformSvc := platform.NewService()
	fyneApp := fyneapp.NewWithID("com.eagleeye.app")
	fyneApp.SetIcon(resources.MustLogo("Logo_Optimal_Gradient.png"))
	desktopApp, ok := fyneApp.(desktop.App)
	if !ok {
		return nil, fmt.Errorf("system tray unsupported on this platform")
	}

	trayWindow := fyneApp.NewWindow(appName)
	if fyneApp.Icon() != nil {
		trayWindow.SetIcon(fyneApp.Icon())
	}
	trayWindowLabel := widget.NewLabel("")
	trayWindow.SetContent(trayWindowLabel)
	trayWindow.SetCloseIntercept(func() {
		trayWindow.Hide()
	})
	trayWindow.Hide()
	desktopApp.SetSystemTrayWindow(trayWindow)

	settings, err := storage.LoadSettings(appName)
	if err != nil {
		logger.Warn("load settings", "error", err)
		settings = preferences.DefaultSettings()
	}

	rt := &AppController{
		ctx:           ctx,
		logger:        logger,
		fyneApp:       fyneApp,
		desktopApp:    desktopApp,
		platformSvc:   platformSvc,
		settings:      settings,
		localizer:     i18n.New(settings.Language),
		state:         newAppState(settings.ShortInterval),
		activeIcon:    resources.MustLogo("Logo_Bright_Gradient.png"),
		pausedIcon:    resources.MustLogo("Logo_Dull_Gradient.png"),
		exerciseCycle: defaultExerciseCycle(),
	}
	settings.Language = rt.localizer.Language()
	rt.settings = settings
	rt.trayLabel = trayWindowLabel
	trayWindowLabel.SetText(rt.localizer.T("main.trayWindowMessage"))

	if err := rt.applyAutostart(exePath, settings.RunOnStartup); err != nil {
		logger.Warn("apply autostart on startup", "error", err)
	}

	rt.keeper = timekeeper.New(rt.settings.TimeKeeperConfig(), timekeeper.Config{TickInterval: time.Second})
	rt.keeper.SetIdleChecker(platform.NewIdleChecker())
	rt.overlayWindow = overlay.New(ctx, fyneApp, overlay.Config{
		Opacity:    opacityToAlpha(settings.OverlayOpacity),
		Fullscreen: settings.Fullscreen,
		Message:    rt.localizer.T("overlay.subtitle"),
	}, nil, rt.localizer)

	animationEngine := animation.New(animation.DefaultConfig(), func(resource fyne.Resource) {
		rt.overlayWindow.SetSprite(resource)
	})
	animationEngine.SetOnExerciseChange(func(exercise animation.ExerciseType) {
		rt.overlayWindow.SetExercise(exercise)
	})
	rt.overlayWindow.SetEngine(animationEngine)
	rt.overlayWindow.SetOnSkip(func() {
		rt.logger.Info("break_skip", "state", "skip")
		rt.overlayWindow.Hide()
		rt.keeper.SkipBreak()
	})

	rt.exerciseSpec = defaultExerciseSpec()
	rt.idleSpec = animation.IdleSpec{
		Open:   rt.exerciseSpec.Center,
		Closed: rt.exerciseSpec.BlinkClosed,
	}

	rt.prefsWindow = preferences.New(fyneApp, rt.settings, preferences.Callbacks{
		OnSave:        rt.savePreferences,
		OnToggleTimer: rt.toggleTimer,
	})
	rt.prefsWindow.SetServiceNotStarted()

	rt.trayManager = tray.New(desktopApp, tray.Callbacks{
		OnPreferences: func() {
			fyne.Do(func() {
				rt.prefsWindow.Show()
			})
		},
		OnTogglePause: rt.togglePauseFromTray,
		OnForceNext:   rt.forceNextBreak,
		OnSkipBreak: func() {
			rt.keeper.SkipBreak()
		},
		OnPauseFor:  rt.pauseFor,
		OnForceLong: rt.forceLongBreak,
		OnQuit: func() {
			rt.keeper.Stop()
			rt.fyneApp.Quit()
		},
	}, rt.localizer)
	desktopApp.SetSystemTrayIcon(rt.activeIcon)

	return rt, nil
}
