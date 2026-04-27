package app

import (
	"eagleeye/internal/core/timekeeper"
	"eagleeye/internal/storage"
	"eagleeye/internal/ui/i18n"
	"eagleeye/internal/ui/overlay"
	"eagleeye/internal/ui/preferences"
	"fmt"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

func (rt *AppController) applyAutostart(exePath string, enabled bool) error {
	if enabled {
		if exePath == "" {
			return fmt.Errorf("apply autostart: executable path is empty")
		}
		return rt.platformSvc.EnableAutostart(appName, exePath)
	}
	return rt.platformSvc.DisableAutostart(appName)
}

func (rt *AppController) startServiceIfNeeded() {
	if rt.state.ServiceStarted() {
		return
	}
	if !rt.settings.BreakTimerStarted {
		rt.settings.BreakTimerStarted = true
		if err := storage.SaveSettings(appName, rt.settings); err != nil {
			rt.logger.Warn("save timer start state", "error", err)
		}
	}
	rt.keeper.Start()
	rt.state.Start()
	rt.desktopApp.SetSystemTrayIcon(rt.activeIcon)
	rt.trayManager.SetPaused(false)
	rt.trayManager.SetStatus(rt.localizer.T("tray.nextBreakIn", formatRemaining(rt.state.NextBreakRemaining())))
	rt.prefsWindow.SetTimerControlState(true)
	rt.prefsWindow.SetServiceRunning(rt.state.NextBreakRemaining())
}

func (rt *AppController) setPauseState(paused bool) {
	if !rt.state.ServiceStarted() {
		return
	}

	if paused {
		rt.keeper.Pause()
		rt.state.SetPaused(true)
		rt.desktopApp.SetSystemTrayIcon(rt.pausedIcon)
		rt.trayManager.SetPaused(true)
		rt.prefsWindow.SetTimerControlState(false)
		rt.prefsWindow.SetServicePaused()
		return
	}

	rt.keeper.Resume()
	rt.state.SetPaused(false)
	rt.desktopApp.SetSystemTrayIcon(rt.activeIcon)
	rt.trayManager.SetPaused(false)
	rt.prefsWindow.SetTimerControlState(true)
	rt.prefsWindow.SetServiceRunning(rt.state.NextBreakRemaining())
}

func (rt *AppController) toggleTimer() {
	if !rt.state.ServiceStarted() {
		rt.startServiceIfNeeded()
		return
	}
	rt.setPauseState(!rt.state.IsPaused())
}

func (rt *AppController) togglePauseFromTray() {
	if !rt.state.ServiceStarted() {
		return
	}
	rt.setPauseState(!rt.state.IsPaused())
}

func (rt *AppController) forceNextBreak() {
	if !rt.state.ServiceStarted() {
		rt.startServiceIfNeeded()
	}
	rt.state.StopPauseTimer()
	if rt.state.IsPaused() {
		rt.setPauseState(false)
	}
	rt.logger.Info("break_force_next", "remaining", rt.state.NextBreakRemaining().String())
	rt.keeper.ForceNextBreak()
}

func (rt *AppController) pauseFor(duration time.Duration) {
	if !rt.state.ServiceStarted() {
		return
	}
	rt.setPauseState(true)
	rt.state.SetPauseTimer(time.AfterFunc(duration, func() {
		fyne.Do(func() {
			rt.setPauseState(false)
		})
	}))
}

func (rt *AppController) forceLongBreak() {
	rt.keeper.ForceBreak(timekeeper.StateLongBreak)
}

func (rt *AppController) savePreferences(updated preferences.Settings) {
	previousSettings := rt.settings
	updated.BreakTimerStarted = rt.settings.BreakTimerStarted || updated.BreakTimerStarted
	languageChanged := i18n.NormalizeLanguage(previousSettings.Language) != i18n.NormalizeLanguage(updated.Language)
	autostartChanged := previousSettings.RunOnStartup != updated.RunOnStartup
	if autostartChanged {
		exePath, err := os.Executable()
		if err != nil {
			rt.showAutostartError(previousSettings, err)
			return
		}
		if err := rt.applyAutostart(exePath, updated.RunOnStartup); err != nil {
			rt.showAutostartError(previousSettings, err)
			return
		}
	}

	rt.settings = updated
	rt.settings.Language = i18n.NormalizeLanguage(rt.settings.Language)
	if err := storage.SaveSettings(appName, rt.settings); err != nil {
		rt.logger.Warn("save settings", "error", err)
	}
	rt.keeper.UpdateConfig(rt.settings.TimeKeeperConfig())
	if languageChanged {
		rt.localizer.SetLanguage(rt.settings.Language)
		rt.trayLabel.SetText(rt.localizer.T("main.trayWindowMessage"))
		if rt.trayManager != nil {
			rt.trayManager.RefreshLocalization()
		}
		rt.overlayWindow.RefreshLocalization()
		rt.prefsWindow.RefreshLocalization()
	}
	rt.overlayWindow.UpdateConfig(overlay.Config{
		Opacity:    opacityToAlpha(rt.settings.OverlayOpacity),
		Fullscreen: rt.settings.Fullscreen,
		Message:    rt.localizer.T("overlay.subtitle"),
	})
}

func (rt *AppController) showAutostartError(previousSettings preferences.Settings, err error) {
	title := rt.localizer.T("prefs.autostartApplyErrorTitle")
	body := rt.localizer.T("prefs.autostartApplyErrorBody", err)
	rt.logger.Warn(title, "error", err)
	dialog.ShowError(fmt.Errorf("%s: %s", title, body), rt.prefsWindow.Window())
	rt.prefsWindow.UpdateSettings(previousSettings)
}
