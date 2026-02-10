package tray

import (
	"fmt"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

// Callbacks defines tray action handlers.
type Callbacks struct {
	OnPreferences func()
	OnTogglePause func()
	OnSkipBreak   func()
	OnPauseFor    func(time.Duration)
	OnForceLong   func()
	OnQuit        func()
}

// Manager handles system tray state.
type Manager struct {
	mu          sync.Mutex
	app         desktop.App
	statusItem  *fyne.MenuItem
	pauseItem   *fyne.MenuItem
	skipItem    *fyne.MenuItem
	pauseFor    *fyne.MenuItem
	forceLong   *fyne.MenuItem
	callbacks   Callbacks
	paused      bool
	inBreak     bool
	statusLabel string
}

// New creates a tray manager with the provided callbacks.
func New(app desktop.App, callbacks Callbacks) *Manager {
	manager := &Manager{
		app:       app,
		callbacks: callbacks,
	}

	manager.statusItem = fyne.NewMenuItem("Status: starting...", nil)
	manager.statusItem.Disabled = true

	preferences := fyne.NewMenuItem("Preferences", func() {
		if manager.callbacks.OnPreferences != nil {
			manager.callbacks.OnPreferences()
		}
	})

	manager.pauseFor = fyne.NewMenuItem("Disable breaks for...", nil)
	manager.pauseFor.ChildMenu = fyne.NewMenu("", fyne.NewMenuItem("5 minutes", func() {
		if manager.callbacks.OnPauseFor != nil {
			manager.callbacks.OnPauseFor(5 * time.Minute)
		}
	}), fyne.NewMenuItem("15 minutes", func() {
		if manager.callbacks.OnPauseFor != nil {
			manager.callbacks.OnPauseFor(15 * time.Minute)
		}
	}), fyne.NewMenuItem("30 minutes", func() {
		if manager.callbacks.OnPauseFor != nil {
			manager.callbacks.OnPauseFor(30 * time.Minute)
		}
	}), fyne.NewMenuItem("60 minutes", func() {
		if manager.callbacks.OnPauseFor != nil {
			manager.callbacks.OnPauseFor(60 * time.Minute)
		}
	}))

	manager.forceLong = fyne.NewMenuItem("Take a long break now", func() {
		if manager.callbacks.OnForceLong != nil {
			manager.callbacks.OnForceLong()
		}
	})

	manager.pauseItem = fyne.NewMenuItem("Pause", func() {
		if manager.callbacks.OnTogglePause != nil {
			manager.callbacks.OnTogglePause()
		}
	})

	manager.skipItem = fyne.NewMenuItem("Skip break", func() {
		if manager.callbacks.OnSkipBreak != nil {
			manager.callbacks.OnSkipBreak()
		}
	})
	manager.skipItem.Disabled = true

	quit := fyne.NewMenuItem("Quit", func() {
		if manager.callbacks.OnQuit != nil {
			manager.callbacks.OnQuit()
		}
	})

	menu := fyne.NewMenu("EagleEye", manager.statusItem, preferences, manager.pauseFor, manager.forceLong, manager.pauseItem, manager.skipItem, quit)
	app.SetSystemTrayMenu(menu)

	return manager
}

// SetStatus updates the status label.
func (manager *Manager) SetStatus(status string) {
	fyne.Do(func() {
		manager.mu.Lock()
		defer manager.mu.Unlock()
		manager.statusLabel = status
		manager.refreshStatusLocked()
	})
}

// SetPaused updates pause state.
func (manager *Manager) SetPaused(paused bool) {
	fyne.Do(func() {
		manager.mu.Lock()
		defer manager.mu.Unlock()
		manager.paused = paused
		if paused {
			manager.pauseItem.Label = "Resume"
		} else {
			manager.pauseItem.Label = "Pause"
		}
		manager.refreshStatusLocked()
	})
}

// SetInBreak toggles break-related menu items.
func (manager *Manager) SetInBreak(inBreak bool) {
	fyne.Do(func() {
		manager.mu.Lock()
		defer manager.mu.Unlock()
		manager.inBreak = inBreak
		manager.skipItem.Disabled = !inBreak
		manager.refreshMenuLocked()
	})
}

func (manager *Manager) refreshStatusLocked() {
	status := manager.statusLabel
	if manager.paused {
		status = fmt.Sprintf("%s (paused)", status)
	}
	manager.statusItem.Label = fmt.Sprintf("Status: %s", status)
	manager.refreshMenuLocked()
}

func (manager *Manager) refreshMenuLocked() {
	if manager.app != nil {
		manager.app.SetSystemTrayMenu(fyne.NewMenu("EagleEye",
			manager.statusItem,
			fyne.NewMenuItem("Preferences", func() {
				if manager.callbacks.OnPreferences != nil {
					manager.callbacks.OnPreferences()
				}
			}),
			manager.pauseFor,
			manager.forceLong,
			manager.pauseItem,
			manager.skipItem,
			fyne.NewMenuItem("Quit", func() {
				if manager.callbacks.OnQuit != nil {
					manager.callbacks.OnQuit()
				}
			}),
		))
	}
}
