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
	"fyne.io/fyne/v2/theme"
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
	statusLine1       *canvas.Text
	statusLine2       *canvas.Text
	statusTimer       *widget.Label
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
	statusIndicator.TextSize = 46
	statusLine1 := canvas.NewText("Service not started", color.NRGBA{R: 200, G: 200, B: 200, A: 255})
	statusLine1.TextSize = 9
	statusLine1.Alignment = fyne.TextAlignCenter
	statusLine2 := canvas.NewText("Close window to start", color.NRGBA{R: 200, G: 200, B: 200, A: 255})
	statusLine2.TextSize = 9
	statusLine2.Alignment = fyne.TextAlignCenter
	statusTimer := widget.NewLabel("")
	statusTimer.Alignment = fyne.TextAlignCenter
	statusBox := container.New(&statusStackLayout{}, statusIndicator, statusLine1, statusLine2, statusTimer)

	heading := canvas.NewText("General", theme.ForegroundColor())
	heading.TextSize = 18
	heading.TextStyle = fyne.TextStyle{Bold: true}
	heading.Alignment = fyne.TextAlignCenter

	labels := map[string]*widget.Label{
		"shortInterval": widget.NewLabel("min"),
		"shortDuration": widget.NewLabel("sec"),
		"longInterval":  widget.NewLabel("min"),
		"longDuration":  widget.NewLabel("min"),
	}
	const valueEntryWidth = float32(60)
	const scheduleLabelWidth = float32(150)

	form := container.NewVBox(
		container.NewCenter(heading),
		newVerticalSpacer(25),
		makeScheduleRow("Short break every", scheduleLabelWidth, shortInt, valueEntryWidth, labels["shortInterval"]),
		makeScheduleRow("Short break duration", scheduleLabelWidth, shortDur, valueEntryWidth, labels["shortDuration"]),
		makeScheduleRow("Long break every", scheduleLabelWidth, longInt, valueEntryWidth, labels["longInterval"]),
		makeScheduleRow("Long break duration", scheduleLabelWidth, longDur, valueEntryWidth, labels["longDuration"]),
		strict,
		idleCheck,
		fullscreen,
		widget.NewLabel("Overlay opacity"),
		opacity,
	)

	saveButton := widget.NewButton("Save", nil)
	cancelButton := widget.NewButton("Cancel", nil)
	saveWrap := container.NewGridWrap(fyne.NewSize(130, 40), saveButton)
	cancelWrap := container.NewGridWrap(fyne.NewSize(130, 40), cancelButton)
	timerToggleButton := widget.NewButton("Pause break timer", nil)
	timerToggleButton.Disable()
	buttons := container.NewHBox(saveWrap, layout.NewSpacer(), cancelWrap)
	footer := container.NewVBox(newVerticalSpacer(15), buttons, timerToggleButton)

	formWithOverlay := container.New(&topRightOverlayLayout{}, form, statusBox)
	content := container.NewBorder(nil, footer, nil, nil, formWithOverlay)
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
		statusLine1:       statusLine1,
		statusLine2:       statusLine2,
		statusTimer:       statusTimer,
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
	prefs.setStatus(color.NRGBA{R: 128, G: 128, B: 128, A: 255}, "Service not started", "Close window to start", "")
	prefs.timerToggleButton.Disable()
}

// SetServiceRunning shows running status with countdown.
func (prefs *Window) SetServiceRunning(remaining time.Duration) {
	prefs.setStatus(color.NRGBA{R: 57, G: 176, B: 99, A: 255}, "Service is running", "Next eye break in", formatDuration(remaining))
	prefs.timerToggleButton.Enable()
}

// SetServicePaused shows paused service status.
func (prefs *Window) SetServicePaused() {
	prefs.setStatus(color.NRGBA{R: 232, G: 190, B: 66, A: 255}, "Service is paused", "Press Resume break timer", "")
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

func (prefs *Window) setStatus(indicator color.NRGBA, line1 string, line2 string, timerText string) {
	fyne.Do(func() {
		prefs.statusIndicator.Color = indicator
		prefs.statusIndicator.Refresh()
		prefs.statusLine1.Text = line1
		prefs.statusLine1.Refresh()
		prefs.statusLine2.Text = line2
		prefs.statusLine2.Refresh()
		prefs.statusTimer.SetText(timerText)
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

func newVerticalSpacer(height float32) fyne.CanvasObject {
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(1, height))
	return spacer
}

type topRightOverlayLayout struct{}

func (layout *topRightOverlayLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}

	objects[0].Move(fyne.NewPos(0, 0))
	objects[0].Resize(size)

	if len(objects) < 2 {
		return
	}

	overlay := objects[1]
	overlaySize := overlay.MinSize()
	const margin = float32(0)
	const overlayWidth = float32(96)
	resizedOverlay := fyne.NewSize(overlayWidth, overlaySize.Height)
	x := size.Width - resizedOverlay.Width - margin
	if x < margin {
		x = margin
	}
	overlay.Move(fyne.NewPos(x, margin))
	overlay.Resize(resizedOverlay)
}

func (layout *topRightOverlayLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.NewSize(0, 0)
	}
	return objects[0].MinSize()
}

type statusStackLayout struct{}

func (layout *statusStackLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 4 {
		return
	}

	indicator := objects[0]
	line1 := objects[1]
	line2 := objects[2]
	timer := objects[3]

	centerX := size.Width / 2
	y := float32(0)

	placeCentered(indicator, centerX, y)
	indicatorSize := indicator.MinSize()
	y += indicatorSize.Height*0.55 + 8

	placeCentered(line1, centerX, y)
	y += line1.MinSize().Height

	placeCentered(line2, centerX, y)
	y += line2.MinSize().Height

	placeCentered(timer, centerX, y)
}

func (layout *statusStackLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.NewSize(0, 0)
	}

	width := float32(0)
	height := float32(0)
	for _, object := range objects {
		size := object.MinSize()
		if size.Width > width {
			width = size.Width
		}
		height += size.Height
	}
	return fyne.NewSize(width, height)
}

func placeCentered(object fyne.CanvasObject, centerX float32, y float32) {
	size := object.MinSize()
	x := centerX - size.Width/2
	object.Move(fyne.NewPos(x, y))
	object.Resize(size)
}

func makeScheduleRow(label string, labelWidth float32, entry *widget.Entry, entryWidth float32, unit *widget.Label) fyne.CanvasObject {
	labelObject := container.NewGridWrap(fyne.NewSize(labelWidth, entry.MinSize().Height), widget.NewLabel(label))
	entryObject := container.NewGridWrap(fyne.NewSize(entryWidth, entry.MinSize().Height), entry)
	return container.NewHBox(labelObject, entryObject, unit)
}
