package app

import (
	"context"
	"eagleeye/internal/core/timekeeper"
	"eagleeye/internal/platform"
	"eagleeye/internal/ui/animation"
	"eagleeye/internal/ui/i18n"
	"eagleeye/internal/ui/overlay"
	"eagleeye/internal/ui/preferences"
	"eagleeye/internal/ui/tray"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// AppController owns the runtime orchestration for one EagleEye process.
type AppController struct {
	ctx context.Context

	logger        *slog.Logger
	fyneApp       fyne.App
	desktopApp    desktop.App
	platformSvc   platform.Service
	settings      preferences.Settings
	localizer     *i18n.Localizer
	state         *appState
	keeper        *timekeeper.TimeKeeper
	overlayWindow *overlay.Window
	trayManager   *tray.Manager
	prefsWindow   *preferences.Window
	trayLabel     *widget.Label

	activeIcon fyne.Resource
	pausedIcon fyne.Resource

	exerciseSpec  animation.ExerciseSpec
	idleSpec      animation.IdleSpec
	exerciseCycle []animation.ExerciseType
}
