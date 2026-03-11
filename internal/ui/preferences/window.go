package preferences

import (
	"fmt"
	"image/color"
	"strconv"
	"time"

	"eagleeye/internal/ui/i18n"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type serviceState int

const (
	serviceStateNotStarted serviceState = iota
	serviceStateRunning
	serviceStatePaused
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
	window    fyne.Window
	settings  Settings
	callbacks Callbacks
	localizer *i18n.Localizer

	labels             map[string]*widget.Label
	scheduleLabels     map[string]*widget.Label
	heading            *canvas.Text
	shortInt           *widget.Entry
	shortDur           *widget.Entry
	longInt            *widget.Entry
	longDur            *widget.Entry
	strict             *widget.Check
	idleCheck          *widget.Check
	opacity            *widget.Slider
	fullscreen         *widget.Check
	languageLabel      *widget.Label
	languageSelect     *widget.Select
	overlayOpacityText *widget.Label
	saveButton         *widget.Button
	cancelButton       *widget.Button

	statusIndicator   *canvas.Text
	statusLine1       *canvas.Text
	statusLine2       *canvas.Text
	statusTimer       *widget.Label
	timerToggleButton *widget.Button

	currentServiceState serviceState
	runningTimerText    string
}

// New creates a preferences window.
func New(app fyne.App, settings Settings, callbacks Callbacks, localizer *i18n.Localizer) *Window {
	if localizer == nil {
		localizer = i18n.New(i18n.LanguageEN)
	}
	window := app.NewWindow(localizer.T("prefs.windowTitle"))
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

	strict := widget.NewCheck("", nil)
	strict.SetChecked(settings.StrictMode)

	idleCheck := widget.NewCheck("", nil)
	idleCheck.SetChecked(settings.IdleEnabled)

	opacity := widget.NewSlider(0.7, 0.95)
	opacity.Value = settings.OverlayOpacity
	opacity.Step = 0.01

	fullscreen := widget.NewCheck("", nil)
	fullscreen.SetChecked(settings.Fullscreen)

	languageSelect := widget.NewSelect(i18n.LanguageOptions(), nil)
	languageSelect.SetSelected(i18n.LanguageDisplayName(settings.Language))

	statusIndicator := canvas.NewText("●", color.NRGBA{R: 128, G: 128, B: 128, A: 255})
	statusIndicator.TextSize = 46
	statusLine1 := canvas.NewText("", color.NRGBA{R: 200, G: 200, B: 200, A: 255})
	statusLine1.TextSize = 9
	statusLine1.Alignment = fyne.TextAlignCenter
	statusLine2 := canvas.NewText("", color.NRGBA{R: 200, G: 200, B: 200, A: 255})
	statusLine2.TextSize = 9
	statusLine2.Alignment = fyne.TextAlignCenter
	statusTimer := widget.NewLabel("")
	statusTimer.Alignment = fyne.TextAlignCenter
	statusBox := container.New(&statusStackLayout{}, statusIndicator, statusLine1, statusLine2, statusTimer)

	heading := canvas.NewText("", theme.ForegroundColor())
	heading.TextSize = 18
	heading.TextStyle = fyne.TextStyle{Bold: true}
	heading.Alignment = fyne.TextAlignCenter

	labels := map[string]*widget.Label{
		"shortInterval": widget.NewLabel(""),
		"shortDuration": widget.NewLabel(""),
		"longInterval":  widget.NewLabel(""),
		"longDuration":  widget.NewLabel(""),
	}
	scheduleLabels := map[string]*widget.Label{
		"shortInterval": widget.NewLabel(""),
		"shortDuration": widget.NewLabel(""),
		"longInterval":  widget.NewLabel(""),
		"longDuration":  widget.NewLabel(""),
	}
	const valueEntryWidth = float32(60)
	const scheduleLabelWidth = float32(190)

	languageLabel := widget.NewLabel("")
	overlayOpacityLabel := widget.NewLabel("")
	form := container.NewVBox(
		newVerticalSpacer(5),
		container.NewCenter(heading),
		newVerticalSpacer(20),
		makeScheduleRow(scheduleLabels["shortInterval"], scheduleLabelWidth, shortInt, valueEntryWidth, labels["shortInterval"]),
		makeScheduleRow(scheduleLabels["shortDuration"], scheduleLabelWidth, shortDur, valueEntryWidth, labels["shortDuration"]),
		makeScheduleRow(scheduleLabels["longInterval"], scheduleLabelWidth, longInt, valueEntryWidth, labels["longInterval"]),
		makeScheduleRow(scheduleLabels["longDuration"], scheduleLabelWidth, longDur, valueEntryWidth, labels["longDuration"]),
		strict,
		idleCheck,
		fullscreen,
		languageLabel,
		languageSelect,
		overlayOpacityLabel,
		opacity,
	)

	saveButton := widget.NewButton("", nil)
	cancelButton := widget.NewButton("", nil)
	saveWrap := container.NewGridWrap(fyne.NewSize(130, 40), saveButton)
	cancelWrap := container.NewGridWrap(fyne.NewSize(130, 40), cancelButton)
	timerToggleButton := widget.NewButton("", nil)
	timerToggleButton.Disable()
	buttons := container.NewHBox(saveWrap, layout.NewSpacer(), cancelWrap)
	footer := container.NewVBox(newVerticalSpacer(15), buttons, timerToggleButton)

	formWithOverlay := container.New(&topRightOverlayLayout{}, form, statusBox)
	content := container.NewBorder(nil, footer, nil, nil, formWithOverlay)
	window.SetContent(content)
	window.Resize(fyne.NewSize(560, 520))
	window.SetFixedSize(true)

	prefs := &Window{
		window:              window,
		settings:            settings,
		callbacks:           callbacks,
		localizer:           localizer,
		labels:              labels,
		scheduleLabels:      scheduleLabels,
		heading:             heading,
		shortInt:            shortInt,
		shortDur:            shortDur,
		longInt:             longInt,
		longDur:             longDur,
		strict:              strict,
		idleCheck:           idleCheck,
		opacity:             opacity,
		fullscreen:          fullscreen,
		languageLabel:       languageLabel,
		languageSelect:      languageSelect,
		overlayOpacityText:  overlayOpacityLabel,
		saveButton:          saveButton,
		cancelButton:        cancelButton,
		statusIndicator:     statusIndicator,
		statusLine1:         statusLine1,
		statusLine2:         statusLine2,
		statusTimer:         statusTimer,
		timerToggleButton:   timerToggleButton,
		currentServiceState: serviceStateNotStarted,
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

	prefs.RefreshLocalization()
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
	prefs.languageSelect.SetSelected(i18n.LanguageDisplayName(settings.Language))
}

// RefreshLocalization refreshes all static UI texts based on the current language.
func (prefs *Window) RefreshLocalization() {
	fyne.Do(func() {
		prefs.window.SetTitle(prefs.localizer.T("prefs.windowTitle"))
		prefs.heading.Text = prefs.localizer.T("prefs.headingGeneral")
		prefs.heading.Refresh()

		prefs.scheduleLabels["shortInterval"].SetText(prefs.localizer.T("prefs.shortBreakEvery"))
		prefs.scheduleLabels["shortDuration"].SetText(prefs.localizer.T("prefs.shortBreakDuration"))
		prefs.scheduleLabels["longInterval"].SetText(prefs.localizer.T("prefs.longBreakEvery"))
		prefs.scheduleLabels["longDuration"].SetText(prefs.localizer.T("prefs.longBreakDuration"))
		prefs.labels["shortInterval"].SetText(prefs.localizer.T("unit.min"))
		prefs.labels["shortDuration"].SetText(prefs.localizer.T("unit.sec"))
		prefs.labels["longInterval"].SetText(prefs.localizer.T("unit.min"))
		prefs.labels["longDuration"].SetText(prefs.localizer.T("unit.min"))

		prefs.strict.Text = prefs.localizer.T("prefs.strictMode")
		prefs.strict.Refresh()
		prefs.idleCheck.Text = prefs.localizer.T("prefs.idleTracking")
		prefs.idleCheck.Refresh()
		prefs.fullscreen.Text = prefs.localizer.T("prefs.fullscreenOverlay")
		prefs.fullscreen.Refresh()
		prefs.languageLabel.SetText(prefs.localizer.T("prefs.language"))
		prefs.overlayOpacityText.SetText(prefs.localizer.T("prefs.overlayOpacity"))
		prefs.saveButton.SetText(prefs.localizer.T("prefs.save"))
		prefs.cancelButton.SetText(prefs.localizer.T("prefs.cancel"))
		prefs.renderServiceStatus()
	})
}

// SetServiceNotStarted shows non-running service status.
func (prefs *Window) SetServiceNotStarted() {
	prefs.currentServiceState = serviceStateNotStarted
	prefs.runningTimerText = ""
	prefs.timerToggleButton.Enable()
	fyne.Do(func() {
		prefs.timerToggleButton.Importance = widget.SuccessImportance
		prefs.timerToggleButton.Refresh()
		prefs.renderServiceStatus()
	})
}

// SetServiceRunning shows running status with countdown.
func (prefs *Window) SetServiceRunning(remaining time.Duration) {
	prefs.currentServiceState = serviceStateRunning
	prefs.runningTimerText = formatDuration(remaining)
	prefs.timerToggleButton.Enable()
	fyne.Do(func() {
		prefs.timerToggleButton.Importance = widget.MediumImportance
		prefs.timerToggleButton.Refresh()
		prefs.renderServiceStatus()
	})
}

// SetServicePaused shows paused service status.
func (prefs *Window) SetServicePaused() {
	prefs.currentServiceState = serviceStatePaused
	prefs.runningTimerText = ""
	prefs.timerToggleButton.Enable()
	fyne.Do(func() {
		prefs.timerToggleButton.Importance = widget.MediumImportance
		prefs.timerToggleButton.Refresh()
		prefs.renderServiceStatus()
	})
}

// SetTimerControlState updates the bottom button label.
func (prefs *Window) SetTimerControlState(isRunning bool) {
	fyne.Do(func() {
		if prefs.currentServiceState == serviceStateNotStarted {
			prefs.timerToggleButton.SetText(prefs.localizer.T("prefs.start"))
			return
		}
		if isRunning {
			prefs.timerToggleButton.SetText(prefs.localizer.T("prefs.pauseBreakTimer"))
		} else {
			prefs.timerToggleButton.SetText(prefs.localizer.T("prefs.resumeBreakTimer"))
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
	settings.Language = i18n.LanguageFromDisplayName(prefs.languageSelect.Selected)

	prefs.settings = settings
	if prefs.callbacks.OnSave != nil {
		prefs.callbacks.OnSave(settings)
	}
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

func (prefs *Window) renderServiceStatus() {
	switch prefs.currentServiceState {
	case serviceStateRunning:
		prefs.statusIndicator.Color = color.NRGBA{R: 57, G: 176, B: 99, A: 255}
		prefs.statusLine1.Text = prefs.localizer.T("prefs.serviceRunningLine")
		prefs.statusLine2.Text = prefs.localizer.T("prefs.nextBreakLine")
		prefs.statusTimer.SetText(prefs.runningTimerText)
		prefs.timerToggleButton.SetText(prefs.localizer.T("prefs.pauseBreakTimer"))
	case serviceStatePaused:
		prefs.statusIndicator.Color = color.NRGBA{R: 232, G: 190, B: 66, A: 255}
		prefs.statusLine1.Text = prefs.localizer.T("prefs.servicePausedLine")
		prefs.statusLine2.Text = prefs.localizer.T("prefs.pressResumeLine")
		prefs.statusTimer.SetText("")
		prefs.timerToggleButton.SetText(prefs.localizer.T("prefs.resumeBreakTimer"))
	default:
		prefs.statusIndicator.Color = color.NRGBA{R: 128, G: 128, B: 128, A: 255}
		prefs.statusLine1.Text = prefs.localizer.T("prefs.serviceNotStartedLine")
		prefs.statusLine2.Text = prefs.localizer.T("prefs.pressStartLine")
		prefs.statusTimer.SetText("")
		prefs.timerToggleButton.SetText(prefs.localizer.T("prefs.start"))
	}

	prefs.statusIndicator.Refresh()
	prefs.statusLine1.Refresh()
	prefs.statusLine2.Refresh()
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
	overlay.Move(fyne.NewPos(x, -6))
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
	y := float32(-5)

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

func makeScheduleRow(label *widget.Label, labelWidth float32, entry *widget.Entry, entryWidth float32, unit *widget.Label) fyne.CanvasObject {
	labelObject := container.NewGridWrap(fyne.NewSize(labelWidth, entry.MinSize().Height), label)
	entryObject := container.NewGridWrap(fyne.NewSize(entryWidth, entry.MinSize().Height), entry)
	return container.NewHBox(labelObject, entryObject, unit)
}
