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

// Run starts the EagleEye desktop application and blocks until the UI exits
func Run(ctx context.Context, args []string) error {
	ctx, cancel := runContext(ctx)
	defer cancel()

	logger, closeLogger := logging.NewJSONLogger(appName)
	defer closeLogger()

	autostartLaunch := IsAutostartLaunch(args)
	guard, err := acquireProcessGuard(logger, autostartLaunch)

	if err != nil {
		return err
	}

	if guard == nil {
		return nil
	}

	defer releaseGuard(logger, guard)

	rt, err := newRuntime(ctx, logger)

	if err != nil {
		return err
	}

	defer rt.keeper.Stop()

	eventWG := rt.startEventLoop()
	defer eventWG.Wait()

	rt.bindActivationHandler(guard)
	rt.quitWhenContextDone()
	rt.showInitialUI(autostartLaunch)

	rt.fyneApp.Run()
	rt.keeper.Stop()
	eventWG.Wait()

	return nil
}

// runContext normalizes nil callers and creates the process cancel scope
func runContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}

	return context.WithCancel(ctx)
}

// acquireProcessGuard enforces single-instance ownership for the process
func acquireProcessGuard(logger *slog.Logger, autostartLaunch bool) (*platform.InstanceGuard, error) {
	guard, err := platform.AcquireSingleInstance(appName)

	if err != nil {
		if errors.Is(err, platform.ErrAlreadyRunning) {
			notifyRunningInstance(logger, autostartLaunch)
			logger.Info("single instance already running")

			return nil, nil
		}

		return nil, fmt.Errorf("acquire single instance: %w", err)
	}

	return guard, nil
}

// notifyRunningInstance asks the existing process to show its UI
func notifyRunningInstance(logger *slog.Logger, autostartLaunch bool) {
	if autostartLaunch {
		return
	}

	if notifyErr := platform.NotifyRunningInstance(appName); notifyErr != nil {
		logger.Warn("notify running instance", "error", notifyErr)
	}
}

// startEventLoop consumes TimeKeeper events until the keeper stops
func (rt *AppController) startEventLoop() *sync.WaitGroup {
	var eventWG sync.WaitGroup
	events := rt.keeper.Subscribe(5)
	eventWG.Add(1)
	go rt.consumeEvents(&eventWG, events)

	return &eventWG
}

// bindActivationHandler opens preferences when another process activates us
func (rt *AppController) bindActivationHandler(guard *platform.InstanceGuard) {
	guard.ListenForActivation(rt.ctx, func() {
		fyne.Do(func() {
			rt.prefsWindow.Show()
		})
	})
}

// quitWhenContextDone stops services and exits Fyne after context cancellation
func (rt *AppController) quitWhenContextDone() {
	go func() {
		<-rt.ctx.Done()

		rt.keeper.Stop()

		fyne.Do(func() {
			rt.fyneApp.Quit()
		})
	}()
}

// showInitialUI either resumes the timer or opens preferences on startup
func (rt *AppController) showInitialUI(autostartLaunch bool) {
	if ShouldStartTimerOnLaunch(rt.settings, autostartLaunch) {
		rt.startServiceIfNeeded()
	} else {
		rt.prefsWindow.Show()
	}
}

// newRuntime builds the controller and wires every UI/runtime dependency
func newRuntime(ctx context.Context, logger *slog.Logger) (*AppController, error) {
	exePath := resolveExecutablePath(logger)
	shell, err := newRuntimeShell()

	if err != nil {
		return nil, err
	}

	settings := loadRuntimeSettings(logger)
	rt := newAppController(ctx, logger, shell, platform.NewService(), settings)

	rt.normalizeSettingsLanguage()
	rt.initializeTrayWindow()
	rt.applyStartupAutostart(exePath)
	rt.initializeTimeKeeper()
	rt.initializeOverlay()
	rt.initializeBreakSpecs()
	rt.initializePreferences()
	rt.initializeTray()

	return rt, nil
}

// runtimeShell groups Fyne shell objects shared by controller setup
type runtimeShell struct {
	fyneApp    fyne.App
	desktopApp desktop.App
	trayLabel  *widget.Label
}

// resolveExecutablePath returns the current executable path if available
func resolveExecutablePath(logger *slog.Logger) string {
	exePath, err := os.Executable()

	if err != nil {
		logger.Warn("resolve executable", "error", err)
	}

	return exePath
}

// newRuntimeShell creates the Fyne app and its system tray window
func newRuntimeShell() (*runtimeShell, error) {
	fyneApp := fyneapp.NewWithID("com.eagleeye.app")
	fyneApp.SetIcon(resources.MustLogo("Logo_Optimal_Gradient.png"))

	desktopApp, ok := fyneApp.(desktop.App)

	if !ok {
		return nil, fmt.Errorf("system tray unsupported on this platform")
	}

	trayLabel := attachSystemTrayWindow(fyneApp, desktopApp)

	return &runtimeShell{
		fyneApp:    fyneApp,
		desktopApp: desktopApp,
		trayLabel:  trayLabel,
	}, nil
}

// attachSystemTrayWindow creates the hidden tray window used by desktop shells
func attachSystemTrayWindow(fyneApp fyne.App, desktopApp desktop.App) *widget.Label {
	trayWindow := fyneApp.NewWindow(appName)

	if fyneApp.Icon() != nil {
		trayWindow.SetIcon(fyneApp.Icon())
	}

	trayLabel := widget.NewLabel("")
	trayWindow.SetContent(trayLabel)
	trayWindow.SetCloseIntercept(func() {
		trayWindow.Hide()
	})
	trayWindow.Hide()

	desktopApp.SetSystemTrayWindow(trayWindow)

	return trayLabel
}

// loadRuntimeSettings reads persisted preferences or falls back to defaults
func loadRuntimeSettings(logger *slog.Logger) preferences.Settings {
	settings, err := storage.LoadSettings(appName)

	if err != nil {
		logger.Warn("load settings", "error", err)

		return preferences.DefaultSettings()
	}

	return settings
}

// newAppController stores runtime dependencies before wiring components
func newAppController(
	ctx context.Context,
	logger *slog.Logger,
	shell *runtimeShell,
	platformSvc platform.Service,
	settings preferences.Settings,
) *AppController {
	return &AppController{
		ctx:           ctx,
		logger:        logger,
		fyneApp:       shell.fyneApp,
		desktopApp:    shell.desktopApp,
		platformSvc:   platformSvc,
		settings:      settings,
		localizer:     i18n.New(settings.Language),
		state:         newAppState(settings.ShortInterval),
		trayLabel:     shell.trayLabel,
		activeIcon:    resources.MustLogo("Logo_Bright_Gradient.png"),
		pausedIcon:    resources.MustLogo("Logo_Dull_Gradient.png"),
		exerciseCycle: defaultExerciseCycle(),
	}
}

// normalizeSettingsLanguage stores the actual language selected by localizer
func (rt *AppController) normalizeSettingsLanguage() {
	rt.settings.Language = rt.localizer.Language()
}

// initializeTrayWindow sets the initial hidden tray window content
func (rt *AppController) initializeTrayWindow() {
	rt.trayLabel.SetText(rt.localizer.T("main.trayWindowMessage"))
}

// applyStartupAutostart synchronizes OS autostart with saved preferences
func (rt *AppController) applyStartupAutostart(exePath string) {
	if err := rt.applyAutostart(exePath, rt.settings.RunOnStartup); err != nil {
		rt.logger.Warn("apply autostart on startup", "error", err)
	}
}

// initializeTimeKeeper creates the timer state machine and idle checker
func (rt *AppController) initializeTimeKeeper() {
	rt.keeper = timekeeper.New(rt.settings.TimeKeeperConfig(), timekeeper.Config{TickInterval: time.Second})
	rt.keeper.SetIdleChecker(platform.NewIdleChecker())
}

// initializeOverlay builds the overlay window and action callbacks
func (rt *AppController) initializeOverlay() {
	rt.overlayWindow = overlay.New(rt.ctx, rt.fyneApp, rt.overlayConfig(), nil, rt.localizer)
	rt.attachAnimationEngine()
	rt.bindOverlayActions()
}

// overlayConfig converts current preferences to overlay config
func (rt *AppController) overlayConfig() overlay.Config {
	return overlay.Config{
		Opacity:    opacityToAlpha(rt.settings.OverlayOpacity),
		Fullscreen: rt.settings.Fullscreen,
		Message:    rt.localizer.T("overlay.subtitle"),
	}
}

// attachAnimationEngine connects animation frames to the overlay sprite
func (rt *AppController) attachAnimationEngine() {
	animationEngine := animation.New(animation.DefaultConfig(), func(resource fyne.Resource) {
		rt.overlayWindow.SetSprite(resource)
	})

	animationEngine.SetOnExerciseChange(func(exercise animation.ExerciseType) {
		rt.overlayWindow.SetExercise(exercise)
	})

	rt.overlayWindow.SetEngine(animationEngine)
}

// bindOverlayActions connects overlay buttons back to the timer
func (rt *AppController) bindOverlayActions() {
	rt.overlayWindow.SetOnSkip(func() {
		rt.logger.Info("break_skip", "state", "skip")
		rt.overlayWindow.Hide()
		rt.keeper.SkipBreak()
	})
}

// initializeBreakSpecs loads sprites used by exercise and idle sessions
func (rt *AppController) initializeBreakSpecs() {
	rt.exerciseSpec = defaultExerciseSpec()
	rt.idleSpec = animation.IdleSpec{
		Open:   rt.exerciseSpec.Center,
		Closed: rt.exerciseSpec.BlinkClosed,
	}
}

// initializePreferences creates the preferences window and initial state
func (rt *AppController) initializePreferences() {
	rt.prefsWindow = preferences.New(rt.fyneApp, rt.settings, rt.preferencesCallbacks())
	rt.prefsWindow.SetServiceNotStarted()
}

// preferencesCallbacks binds preferences actions to controller methods
func (rt *AppController) preferencesCallbacks() preferences.Callbacks {
	return preferences.Callbacks{
		OnSave:        rt.savePreferences,
		OnToggleTimer: rt.toggleTimer,
	}
}

// initializeTray creates tray menu state and icon
func (rt *AppController) initializeTray() {
	rt.trayManager = tray.New(rt.desktopApp, rt.trayCallbacks(), rt.localizer)
	rt.desktopApp.SetSystemTrayIcon(rt.activeIcon)
}

// trayCallbacks binds tray menu actions to controller methods
func (rt *AppController) trayCallbacks() tray.Callbacks {
	return tray.Callbacks{
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
	}
}
