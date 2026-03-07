package tray

import (
	"fmt"
	"sync"
	"time"

	"eagleeye/internal/ui/i18n"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/systray"
)

// Callbacks defines tray action handlers.
type Callbacks struct {
	OnPreferences func()
	OnTogglePause func()
	OnForceNext   func()
	OnSkipBreak   func()
	OnPauseFor    func(time.Duration)
	OnForceLong   func()
	OnQuit        func()
}

// Manager handles system tray state.
type Manager struct {
	mu sync.Mutex

	app       desktop.App
	callbacks Callbacks
	localizer *i18n.Localizer

	statusItem      *fyne.MenuItem
	forceNextItem   *fyne.MenuItem
	preferencesItem *fyne.MenuItem
	pauseItem       *fyne.MenuItem
	skipItem        *fyne.MenuItem
	pauseForItem    *fyne.MenuItem
	pause5Item      *fyne.MenuItem
	pause15Item     *fyne.MenuItem
	pause30Item     *fyne.MenuItem
	pause60Item     *fyne.MenuItem
	forceLongItem   *fyne.MenuItem
	quitItem        *fyne.MenuItem

	paused      bool
	inBreak     bool
	statusLabel string

	tooltipEnabled bool
}

// New creates a tray manager with the provided callbacks.
func New(app desktop.App, callbacks Callbacks, localizer *i18n.Localizer) *Manager {
	if localizer == nil {
		localizer = i18n.New(i18n.LanguageEN)
	}

	manager := &Manager{
		app:       app,
		callbacks: callbacks,
		localizer: localizer,
	}

	manager.statusItem = fyne.NewMenuItem("", nil)
	manager.statusItem.Disabled = true

	manager.forceNextItem = fyne.NewMenuItem("", func() {
		if manager.callbacks.OnForceNext != nil {
			manager.callbacks.OnForceNext()
		}
	})

	manager.preferencesItem = fyne.NewMenuItem("", func() {
		if manager.callbacks.OnPreferences != nil {
			manager.callbacks.OnPreferences()
		}
	})

	manager.pause5Item = fyne.NewMenuItem("", func() {
		if manager.callbacks.OnPauseFor != nil {
			manager.callbacks.OnPauseFor(5 * time.Minute)
		}
	})
	manager.pause15Item = fyne.NewMenuItem("", func() {
		if manager.callbacks.OnPauseFor != nil {
			manager.callbacks.OnPauseFor(15 * time.Minute)
		}
	})
	manager.pause30Item = fyne.NewMenuItem("", func() {
		if manager.callbacks.OnPauseFor != nil {
			manager.callbacks.OnPauseFor(30 * time.Minute)
		}
	})
	manager.pause60Item = fyne.NewMenuItem("", func() {
		if manager.callbacks.OnPauseFor != nil {
			manager.callbacks.OnPauseFor(60 * time.Minute)
		}
	})

	manager.pauseForItem = fyne.NewMenuItem("", nil)
	manager.pauseForItem.ChildMenu = fyne.NewMenu("", manager.pause5Item, manager.pause15Item, manager.pause30Item, manager.pause60Item)

	manager.forceLongItem = fyne.NewMenuItem("", func() {
		if manager.callbacks.OnForceLong != nil {
			manager.callbacks.OnForceLong()
		}
	})

	manager.pauseItem = fyne.NewMenuItem("", func() {
		if manager.callbacks.OnTogglePause != nil {
			manager.callbacks.OnTogglePause()
		}
	})

	manager.skipItem = fyne.NewMenuItem("", func() {
		if manager.callbacks.OnSkipBreak != nil {
			manager.callbacks.OnSkipBreak()
		}
	})
	manager.skipItem.Disabled = true

	manager.quitItem = fyne.NewMenuItem("", func() {
		if manager.callbacks.OnQuit != nil {
			manager.callbacks.OnQuit()
		}
	})

	manager.statusLabel = manager.localizer.T("tray.statusStarting")
	manager.refreshLocalizationLocked()
	manager.refreshMenuLocked()
	return manager
}

// RefreshLocalization updates tray texts after language changes.
func (manager *Manager) RefreshLocalization() {
	fyne.Do(func() {
		manager.mu.Lock()
		defer manager.mu.Unlock()
		manager.refreshLocalizationLocked()
		manager.refreshMenuLocked()
	})
}

// SetStatus updates the status label.
func (manager *Manager) SetStatus(status string) {
	fyne.Do(func() {
		manager.mu.Lock()
		defer manager.mu.Unlock()
		manager.statusLabel = status
		manager.tooltipEnabled = true
		manager.refreshStatusLocked()
		manager.refreshMenuLocked()
	})
}

// SetPaused updates pause state.
func (manager *Manager) SetPaused(paused bool) {
	fyne.Do(func() {
		manager.mu.Lock()
		defer manager.mu.Unlock()
		manager.paused = paused
		manager.refreshLocalizationLocked()
		manager.refreshMenuLocked()
	})
}

// SetInBreak toggles break-related menu items.
func (manager *Manager) SetInBreak(inBreak bool) {
	fyne.Do(func() {
		manager.mu.Lock()
		defer manager.mu.Unlock()
		manager.inBreak = inBreak
		manager.forceNextItem.Disabled = inBreak
		manager.skipItem.Disabled = !inBreak
		manager.refreshMenuLocked()
	})
}

func (manager *Manager) refreshLocalizationLocked() {
	manager.forceNextItem.Label = manager.localizer.T("tray.takeNextBreakNow")
	manager.preferencesItem.Label = manager.localizer.T("tray.preferences")
	manager.pauseForItem.Label = manager.localizer.T("tray.disableBreaksFor")
	manager.pause5Item.Label = manager.localizer.T("tray.pauseForMinutes", 5)
	manager.pause15Item.Label = manager.localizer.T("tray.pauseForMinutes", 15)
	manager.pause30Item.Label = manager.localizer.T("tray.pauseForMinutes", 30)
	manager.pause60Item.Label = manager.localizer.T("tray.pauseForMinutes", 60)
	manager.forceLongItem.Label = manager.localizer.T("tray.takeLongBreakNow")
	if manager.paused {
		manager.pauseItem.Label = manager.localizer.T("tray.resume")
	} else {
		manager.pauseItem.Label = manager.localizer.T("tray.pause")
	}
	manager.skipItem.Label = manager.localizer.T("tray.skipBreak")
	manager.quitItem.Label = manager.localizer.T("tray.quit")
	manager.refreshStatusLocked()
}

func (manager *Manager) refreshStatusLocked() {
	status := manager.statusLabel
	if manager.paused {
		status = fmt.Sprintf("%s %s", status, manager.localizer.T("tray.pausedSuffix"))
	}
	manager.statusItem.Label = manager.localizer.T("tray.statusFormat", status)
	if manager.tooltipEnabled {
		systray.SetTooltip(manager.statusItem.Label)
	}
}

func (manager *Manager) refreshMenuLocked() {
	if manager.app == nil {
		return
	}
	manager.app.SetSystemTrayMenu(fyne.NewMenu(
		manager.localizer.T("tray.menuTitle"),
		manager.statusItem,
		manager.forceNextItem,
		manager.preferencesItem,
		manager.pauseForItem,
		manager.forceLongItem,
		manager.pauseItem,
		manager.skipItem,
		manager.quitItem,
	))
}
