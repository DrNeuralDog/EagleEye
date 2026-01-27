package preferences

import (
	"fmt"
	"image/color"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// Callbacks defines preferences window actions.
type Callbacks struct {
	OnSave        func(Settings)
	OnCancel      func()
	OnDismiss     func()
	OnToggleTimer func()
}

// Window handles the preferences UI.
type Window struct {
	window            fyne.Window
	settings          Settings
	callbacks         Callbacks
	labels            map[string]*widget.Label
	shortInt          *widget.Entry
	shortDur          *widget.Entry
	longInt           *widget.Entry
	longDur           *widget.Entry
	strict            *widget.Check
	idleCheck         *widget.Check
	opacity           *widget.Slider
	fullscreen        *widget.Check
	statusIndicator   *canvas.Text
	statusDescription *widget.Label
	timerToggleButton *widget.Button
}

// New creates a preferences window.
func New(app fyne.App, settings Settings, callbacks Callbacks) *Window {
	window := app.NewWindow("EagleEye Settings")
	if app.Icon() != nil {
		window.SetIcon(app.Icon())
	}

	shortInt := widget.NewEntry()
	shortDur := widget.NewEntry()
	longInt := widget.NewEntry()
	longDur := widget.NewEntry()

	shortInt.SetText(fmt.Sprintf("%d", int(settings.ShortInterval.Minutes())))
	shortDur.SetText(fmt.Sprintf("%d", int(settings.ShortDuration.Seconds())))
	longInt.SetText(fmt.Sprintf("%d", int(settings.LongInterval.Minutes())))
	longDur.SetText(fmt.Sprintf("%d", int(settings.LongDuration.Minutes())))

	strict := widget.NewCheck("Strict mode (disable skip)", nil)
	strict.SetChecked(settings.StrictMode)

	idleCheck := widget.NewCheck("Enable idle tracking", nil)
	idleCheck.SetChecked(settings.IdleEnabled)

	opacity := widget.NewSlider(0.7, 0.95)
	opacity.Value = settings.OverlayOpacity
	opacity.Step = 0.01

	fullscreen := widget.NewCheck("Fullscreen overlay", nil)
	fullscreen.SetChecked(settings.Fullscreen)

	statusIndicator := canvas.NewText("●", color.NRGBA{R: 128, G: 128, B: 128, A: 255})
	statusIndicator.TextSize = 16
	statusDescription := widget.NewLabel("Service is not started yet.\nClose this window to start.")
	statusDescription.Alignment = fyne.TextAlignTrailing
	statusBox := container.NewVBox(
		container.NewHBox(layout.NewSpacer(), statusIndicator),
		statusDescription,
	)

	labels := map[string]*widget.Label{
		"shortInterval": widget.NewLabel("min"),
		"shortDuration": widget.NewLabel("sec"),
		"longInterval":  widget.NewLabel("min"),
		"longDuration":  widget.NewLabel("min"),
	}

	form := container.NewVBox(
		container.NewHBox(layout.NewSpacer(), statusBox),
		widget.NewLabelWithStyle("General", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewHBox(widget.NewLabel("Short break every"), shortInt, labels["shortInterval"]),
		container.NewHBox(widget.NewLabel("Short break duration"), shortDur, labels["shortDuration"]),
		container.NewHBox(widget.NewLabel("Long break every"), longInt, labels["longInterval"]),
		container.NewHBox(widget.NewLabel("Long break duration"), longDur, labels["longDuration"]),
		strict,
		idleCheck,
		widget.NewLabel("Overlay opacity"),
		opacity,
		fullscreen,
	)

	saveButton := widget.NewButton("Save", nil)
	cancelButton := widget.NewButton("Cancel", nil)
	timerToggleButton := widget.NewButton("Pause break timer", nil)
	timerToggleButton.Disable()
	buttons := container.NewHBox(saveButton, layout.NewSpacer(), cancelButton)
	footer := container.NewVBox(buttons, timerToggleButton)

	content := container.NewBorder(nil, footer, nil, nil, form)
	window.SetContent(content)
	window.Resize(fyne.NewSize(520, 500))

	prefs := &Window{
		window:            window,
		settings:          settings,
		callbacks:         callbacks,
		labels:            labels,
		shortInt:          shortInt,
		shortDur:          shortDur,
		longInt:           longInt,
		longDur:           longDur,
		strict:            strict,
		idleCheck:         idleCheck,
		opacity:           opacity,
		fullscreen:        fullscreen,
		statusIndicator:   statusIndicator,
		statusDescription: statusDescription,
		timerToggleButton: timerToggleButton,
	}

	saveButton.OnTapped = prefs.handleSave
	cancelButton.OnTapped = func() {
		prefs.dismiss(false)
	}
	timerToggleButton.OnTapped = func() {
		if prefs.callbacks.OnToggleTimer != nil {
			prefs.callbacks.OnToggleTimer()
		}
	}
	window.SetCloseIntercept(func() {
		prefs.dismiss(false)
	})

	prefs.SetServiceNotStarted()

	return prefs
}

// Show displays the preferences window.
func (prefs *Window) Show() {
	prefs.window.Show()
	prefs.window.RequestFocus()
}

// UpdateSettings replaces window values.
func (prefs *Window) UpdateSettings(settings Settings) {
	prefs.settings = settings
	prefs.shortInt.SetText(fmt.Sprintf("%d", int(settings.ShortInterval.Minutes())))
	prefs.shortDur.SetText(fmt.Sprintf("%d", int(settings.ShortDuration.Seconds())))
	prefs.longInt.SetText(fmt.Sprintf("%d", int(settings.LongInterval.Minutes())))
	prefs.longDur.SetText(fmt.Sprintf("%d", int(settings.LongDuration.Minutes())))
	prefs.strict.SetChecked(settings.StrictMode)
	prefs.idleCheck.SetChecked(settings.IdleEnabled)
	prefs.opacity.Value = settings.OverlayOpacity
	prefs.opacity.Refresh()
	prefs.fullscreen.SetChecked(settings.Fullscreen)
}

// SetServiceNotStarted shows non-running service status.
func (prefs *Window) SetServiceNotStarted() {
	prefs.setStatus(color.NRGBA{R: 128, G: 128, B: 128, A: 255}, "Service is not started yet.\nClose this window to start.")
	prefs.timerToggleButton.Disable()
}

// SetServiceRunning shows running status with countdown.
func (prefs *Window) SetServiceRunning(remaining time.Duration) {
	prefs.setStatus(color.NRGBA{R: 57, G: 176, B: 99, A: 255}, fmt.Sprintf("Service is running.\nNext eye break in %s", formatDuration(remaining)))
	prefs.timerToggleButton.Enable()
}

// SetServicePaused shows paused service status.
func (prefs *Window) SetServicePaused() {
	prefs.setStatus(color.NRGBA{R: 128, G: 128, B: 128, A: 255}, "Service is paused.\nPress Resume break timer.")
	prefs.timerToggleButton.Enable()
}

// SetTimerControlState updates the bottom button label.
func (prefs *Window) SetTimerControlState(isRunning bool) {
	fyne.Do(func() {
		if isRunning {
			prefs.timerToggleButton.SetText("Pause break timer")
		} else {
			prefs.timerToggleButton.SetText("Resume break timer")
		}
	})
}

func (prefs *Window) handleSave() {
	settings := prefs.settings

	if minutes, ok := parsePositiveInt(prefs.shortInt.Text); ok {
		settings.ShortInterval = time.Duration(minutes) * time.Minute
	}
	if seconds, ok := parsePositiveInt(prefs.shortDur.Text); ok {
		settings.ShortDuration = time.Duration(seconds) * time.Second
	}
	if minutes, ok := parsePositiveInt(prefs.longInt.Text); ok {
		settings.LongInterval = time.Duration(minutes) * time.Minute
	}
	if minutes, ok := parsePositiveInt(prefs.longDur.Text); ok {
		settings.LongDuration = time.Duration(minutes) * time.Minute
	}

	settings.StrictMode = prefs.strict.Checked
	settings.IdleEnabled = prefs.idleCheck.Checked
	settings.OverlayOpacity = prefs.opacity.Value
	settings.Fullscreen = prefs.fullscreen.Checked

	prefs.settings = settings
	if prefs.callbacks.OnSave != nil {
		prefs.callbacks.OnSave(settings)
	}
	prefs.dismiss(true)
}

func parsePositiveInt(value string) (int, bool) {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, false
	}
	return parsed, true
}

func (prefs *Window) dismiss(saved bool) {
	prefs.window.Hide()
	if !saved && prefs.callbacks.OnCancel != nil {
		prefs.callbacks.OnCancel()
	}
	if prefs.callbacks.OnDismiss != nil {
		prefs.callbacks.OnDismiss()
	}
}

func (prefs *Window) setStatus(indicator color.NRGBA, text string) {
	fyne.Do(func() {
		prefs.statusIndicator.Color = indicator
		prefs.statusIndicator.Refresh()
		prefs.statusDescription.SetText(text)
	})
}

func formatDuration(value time.Duration) string {
	if value < 0 {
		value = 0
	}
	seconds := int(value.Seconds())
	minutes := seconds / 60
	seconds = seconds % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}
