package preferences

import (
	"fmt"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// Window handles the preferences UI.
type Window struct {
	window    fyne.Window
	settings  Settings
	onSave    func(Settings)
	onCancel  func()
	labels    map[string]*widget.Label
	shortInt  *widget.Entry
	shortDur  *widget.Entry
	longInt   *widget.Entry
	longDur   *widget.Entry
	strict    *widget.Check
	idleCheck *widget.Check
	opacity   *widget.Slider
	fullscreen *widget.Check
}

// New creates a preferences window.
func New(app fyne.App, settings Settings, onSave func(Settings)) *Window {
	window := app.NewWindow("EagleEye Settings")

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

	labels := map[string]*widget.Label{
		"shortInterval": widget.NewLabel("min"),
		"shortDuration": widget.NewLabel("sec"),
		"longInterval":  widget.NewLabel("min"),
		"longDuration":  widget.NewLabel("min"),
	}

	form := container.NewVBox(
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
	buttons := container.NewHBox(saveButton, layout.NewSpacer(), cancelButton)

	content := container.NewBorder(nil, buttons, nil, nil, form)
	window.SetContent(content)
	window.Resize(fyne.NewSize(420, 420))

	prefs := &Window{
		window:    window,
		settings:  settings,
		onSave:    onSave,
		labels:    labels,
		shortInt:  shortInt,
		shortDur:  shortDur,
		longInt:   longInt,
		longDur:   longDur,
		strict:    strict,
		idleCheck: idleCheck,
		opacity:   opacity,
		fullscreen: fullscreen,
	}

	saveButton.OnTapped = prefs.handleSave
	cancelButton.OnTapped = func() {
		window.Hide()
		if prefs.onCancel != nil {
			prefs.onCancel()
		}
	}

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
	if prefs.onSave != nil {
		prefs.onSave(settings)
	}
	prefs.window.Hide()
}

func parsePositiveInt(value string) (int, bool) {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, false
	}
	return parsed, true
}
