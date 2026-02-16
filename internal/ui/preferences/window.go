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

const (
	statusBarMainTextSize  = float32(11)
	statusTimerTextSize    = float32(12) // slightly larger than main status line
	statusIndicatorSize    = float32(18)
	statusBarFromFooterGap = float32(2) // space between timer button and divider line

	prefsWindowWidth          = float32(560)
	scheduleLabelWidthEN      = float32(190)
	scheduleLabelExtraRU      = prefsWindowWidth * 0.2 // ~20% of fixed preferences width
	valueEntryWidth           = float32(60)
	languageSelectWrapWidth   = float32(142)
	strictModeTopSpacerHeight = float32(8)
	saveCancelButtonWidth     = float32(130 * 1.4)
	saveCancelButtonHeight    = float32(40)
	// Vertical gap between run-on-startup, language row, and overlay opacity label.
	preferencesMidFormGap = float32(12)
)

var (
	statusHintColor     = color.NRGBA{R: 200, G: 200, B: 200, A: 255}
	statusTimerColor    = color.NRGBA{R: 252, G: 252, B: 252, A: 255}
	statusBarDividerClr = color.NRGBA{R: 72, G: 72, B: 72, A: 255}
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
	callbacks    Callbacks
	uiLocalizer *i18n.Localizer

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
	runOnStartup       *widget.Check
	languageLabel      *widget.Label
	languageSelect     *widget.Select
	overlayOpacityText *widget.Label
	saveButton         *widget.Button
	cancelButton       *widget.Button

	statusIndicatorDot *canvas.Circle
	statusBarMain      *canvas.Text
	statusBarTimer     *canvas.Text
	timerToggleButton  *widget.Button

	scheduleSection      *fyne.Container
	scheduleLayoutLang   string

	currentServiceState serviceState
	runningTimerText    string
}

// New creates a preferences window.
func New(app fyne.App, settings Settings, callbacks Callbacks) *Window {
	uiLocalizer := i18n.New(settings.Language)
	window := app.NewWindow(uiLocalizer.T("prefs.windowTitle"))
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

	runOnStartup := widget.NewCheck("", nil)
	runOnStartup.SetChecked(settings.RunOnStartup)

	languageSelect := widget.NewSelect(i18n.LanguageOptions(), nil)
	languageSelect.SetSelected(i18n.LanguageDisplayName(settings.Language))

	statusDot := canvas.NewCircle(color.NRGBA{R: 128, G: 128, B: 128, A: 255})
	statusDot.StrokeWidth = 0
	statusIndicatorWrap := container.NewGridWrap(fyne.NewSize(statusIndicatorSize, statusIndicatorSize), statusDot)
	statusBarMain := canvas.NewText("", statusHintColor)
	statusBarMain.TextSize = statusBarMainTextSize
	statusBarMain.Alignment = fyne.TextAlignLeading
	statusBarTimer := canvas.NewText("", statusTimerColor)
	statusBarTimer.TextSize = statusTimerTextSize
	statusBarTimer.Alignment = fyne.TextAlignLeading
	statusTextLine := container.NewHBox(statusBarMain, statusBarTimer)
	statusTextArea := container.NewMax(statusTextLine)
	indicatorCell := container.NewVBox(layout.NewSpacer(), statusIndicatorWrap, layout.NewSpacer())
	// Center = full-width text (leading); right = indicator pinned to window/content right edge.
	statusRow := container.NewBorder(nil, nil, nil, indicatorCell, statusTextArea)
	statusDivider := canvas.NewRectangle(statusBarDividerClr)
	statusDivider.SetMinSize(fyne.NewSize(1, 1))
	statusBar := container.NewBorder(statusDivider, nil, nil, nil, container.NewPadded(statusRow))

	heading := canvas.NewText("", theme.ForegroundColor())
	heading.TextSize = 19
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

	initLang := i18n.NormalizeLanguage(settings.Language)
	initLabelWidth := scheduleLabelWidthForLang(initLang)
	scheduleRows := buildScheduleRowGroup(scheduleLabels, labels, initLabelWidth, shortInt, shortDur, longInt, longDur)
	scheduleSection := container.NewVBox(scheduleRows...)

	languageLabel := widget.NewLabel("")
	languageSelectWrap := container.NewGridWrap(
		fyne.NewSize(languageSelectWrapWidth, languageSelect.MinSize().Height),
		languageSelect,
	)
	languageRow := container.NewHBox(languageLabel, languageSelectWrap, layout.NewSpacer())

	overlayOpacityLabel := widget.NewLabel("")
	form := container.NewVBox(
		newVerticalSpacer(5),
		container.NewCenter(heading),
		newVerticalSpacer(20),
		scheduleSection,
		newVerticalSpacer(strictModeTopSpacerHeight),
		strict,
		idleCheck,
		fullscreen,
		runOnStartup,
		newVerticalSpacer(preferencesMidFormGap),
		languageRow,
		newVerticalSpacer(preferencesMidFormGap),
		overlayOpacityLabel,
		opacity,
	)

	saveButton := widget.NewButton("", nil)
	cancelButton := widget.NewButton("", nil)
	saveWrap := container.NewGridWrap(fyne.NewSize(saveCancelButtonWidth, saveCancelButtonHeight), saveButton)
	cancelWrap := container.NewGridWrap(fyne.NewSize(saveCancelButtonWidth, saveCancelButtonHeight), cancelButton)
	timerToggleButton := widget.NewButton("", nil)
	timerToggleButton.Disable()
	buttons := container.NewHBox(saveWrap, layout.NewSpacer(), cancelWrap)
	footer := container.NewVBox(newVerticalSpacer(15), buttons, timerToggleButton)

	center := container.NewVBox(form, footer, newVerticalSpacer(statusBarFromFooterGap))
	content := container.NewBorder(nil, statusBar, nil, nil, center)
	window.SetContent(content)
	window.Resize(fyne.NewSize(prefsWindowWidth, 520))
	window.SetFixedSize(true)

	prefs := &Window{
		window:              window,
		settings:            settings,
		callbacks:           callbacks,
		uiLocalizer:         uiLocalizer,
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
		runOnStartup:        runOnStartup,
		languageLabel:       languageLabel,
		languageSelect:      languageSelect,
		overlayOpacityText:  overlayOpacityLabel,
		saveButton:          saveButton,
		cancelButton:        cancelButton,
		statusIndicatorDot:  statusDot,
		statusBarMain:       statusBarMain,
		statusBarTimer:      statusBarTimer,
		timerToggleButton:   timerToggleButton,
		scheduleSection:     scheduleSection,
		scheduleLayoutLang:  initLang,
		currentServiceState: serviceStateNotStarted,
	}

	saveButton.OnTapped = prefs.handleSave
	prefs.languageSelect.OnChanged = func(string) {
		prefs.uiLocalizer.SetLanguage(i18n.LanguageFromDisplayName(prefs.languageSelect.Selected))
		prefs.RefreshLocalization()
	}
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

// Window returns the underlying fyne window.
func (prefs *Window) Window() fyne.Window {
	return prefs.window
}

// UpdateSettings replaces window values.
func (prefs *Window) UpdateSettings(settings Settings) {
	prefs.settings = settings
	prefs.uiLocalizer.SetLanguage(settings.Language)
	prefs.shortInt.SetText(fmt.Sprintf("%d", int(settings.ShortInterval.Minutes())))
	prefs.shortDur.SetText(fmt.Sprintf("%d", int(settings.ShortDuration.Seconds())))
	prefs.longInt.SetText(fmt.Sprintf("%d", int(settings.LongInterval.Minutes())))
	prefs.longDur.SetText(fmt.Sprintf("%d", int(settings.LongDuration.Minutes())))
	prefs.strict.SetChecked(settings.StrictMode)
	prefs.idleCheck.SetChecked(settings.IdleEnabled)
	prefs.opacity.Value = settings.OverlayOpacity
	prefs.opacity.Refresh()
	prefs.fullscreen.SetChecked(settings.Fullscreen)
	prefs.runOnStartup.SetChecked(settings.RunOnStartup)
	prefs.languageSelect.SetSelected(i18n.LanguageDisplayName(settings.Language))
	prefs.RefreshLocalization()
}

// RefreshLocalization refreshes all static UI texts based on the current language.
func (prefs *Window) RefreshLocalization() {
	fyne.Do(func() {
		prefs.window.SetTitle(prefs.uiLocalizer.T("prefs.windowTitle"))
		prefs.heading.Text = prefs.uiLocalizer.T("prefs.headingGeneral")
		prefs.heading.Refresh()

		prefs.scheduleLabels["shortInterval"].SetText(prefs.uiLocalizer.T("prefs.shortBreakEvery"))
		prefs.scheduleLabels["shortDuration"].SetText(prefs.uiLocalizer.T("prefs.shortBreakDuration"))
		prefs.scheduleLabels["longInterval"].SetText(prefs.uiLocalizer.T("prefs.longBreakEvery"))
		prefs.scheduleLabels["longDuration"].SetText(prefs.uiLocalizer.T("prefs.longBreakDuration"))
		prefs.labels["shortInterval"].SetText(prefs.uiLocalizer.T("unit.min"))
		prefs.labels["shortDuration"].SetText(prefs.uiLocalizer.T("unit.sec"))
		prefs.labels["longInterval"].SetText(prefs.uiLocalizer.T("unit.min"))
		prefs.labels["longDuration"].SetText(prefs.uiLocalizer.T("unit.min"))

		prefs.strict.Text = prefs.uiLocalizer.T("prefs.strictMode")
		prefs.strict.Refresh()
		prefs.idleCheck.Text = prefs.uiLocalizer.T("prefs.idleTracking")
		prefs.idleCheck.Refresh()
		prefs.fullscreen.Text = prefs.uiLocalizer.T("prefs.fullscreenOverlay")
		prefs.fullscreen.Refresh()
		prefs.runOnStartup.Text = prefs.uiLocalizer.T("prefs.runOnStartup")
		prefs.runOnStartup.Refresh()
		prefs.languageLabel.SetText(prefs.uiLocalizer.T("prefs.language"))
		prefs.overlayOpacityText.SetText(prefs.uiLocalizer.T("prefs.overlayOpacity"))
		prefs.saveButton.SetText(prefs.uiLocalizer.T("prefs.save"))
		prefs.cancelButton.SetText(prefs.uiLocalizer.T("prefs.cancel"))
		prefs.renderServiceStatus()
		prefs.refreshScheduleLayoutIfNeeded()
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
			prefs.timerToggleButton.SetText(prefs.uiLocalizer.T("prefs.start"))
			return
		}
		if isRunning {
			prefs.timerToggleButton.SetText(prefs.uiLocalizer.T("prefs.pauseBreakTimer"))
		} else {
			prefs.timerToggleButton.SetText(prefs.uiLocalizer.T("prefs.resumeBreakTimer"))
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
	settings.RunOnStartup = prefs.runOnStartup.Checked
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
	if !saved {
		prefs.uiLocalizer.SetLanguage(prefs.settings.Language)
		prefs.languageSelect.SetSelected(i18n.LanguageDisplayName(prefs.settings.Language))
		prefs.RefreshLocalization()
	}
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
		prefs.statusIndicatorDot.FillColor = color.NRGBA{R: 57, G: 176, B: 99, A: 255}
		prefs.statusBarMain.Text = fmt.Sprintf("%s — %s ",
			prefs.uiLocalizer.T("prefs.serviceRunningLine"),
			prefs.uiLocalizer.T("prefs.nextBreakLine"),
		)
		prefs.statusBarTimer.Text = prefs.runningTimerText
		prefs.timerToggleButton.SetText(prefs.uiLocalizer.T("prefs.pauseBreakTimer"))
	case serviceStatePaused:
		prefs.statusIndicatorDot.FillColor = color.NRGBA{R: 232, G: 190, B: 66, A: 255}
		prefs.statusBarMain.Text = fmt.Sprintf("%s — %s",
			prefs.uiLocalizer.T("prefs.servicePausedLine"),
			prefs.uiLocalizer.T("prefs.pressResumeLine"),
		)
		prefs.statusBarTimer.Text = ""
		prefs.timerToggleButton.SetText(prefs.uiLocalizer.T("prefs.resumeBreakTimer"))
	default:
		prefs.statusIndicatorDot.FillColor = color.NRGBA{R: 128, G: 128, B: 128, A: 255}
		prefs.statusBarMain.Text = fmt.Sprintf("%s — %s",
			prefs.uiLocalizer.T("prefs.serviceNotStartedLine"),
			prefs.uiLocalizer.T("prefs.pressStartLine"),
		)
		prefs.statusBarTimer.Text = ""
		prefs.timerToggleButton.SetText(prefs.uiLocalizer.T("prefs.start"))
	}

	prefs.statusIndicatorDot.Refresh()
	prefs.statusBarMain.Refresh()
	prefs.statusBarTimer.Refresh()
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

func makeScheduleRow(label *widget.Label, labelWidth float32, entry *widget.Entry, entryWidth float32, unit *widget.Label) fyne.CanvasObject {
	labelObject := container.NewGridWrap(fyne.NewSize(labelWidth, entry.MinSize().Height), label)
	entryObject := container.NewGridWrap(fyne.NewSize(entryWidth, entry.MinSize().Height), entry)
	return container.NewHBox(labelObject, entryObject, unit)
}

func scheduleLabelWidthForLang(lang string) float32 {
	if i18n.NormalizeLanguage(lang) == i18n.LanguageRU {
		return scheduleLabelWidthEN + scheduleLabelExtraRU
	}
	return scheduleLabelWidthEN
}

func buildScheduleRowGroup(
	scheduleLabels map[string]*widget.Label,
	labels map[string]*widget.Label,
	labelWidth float32,
	shortInt, shortDur, longInt, longDur *widget.Entry,
) []fyne.CanvasObject {
	return []fyne.CanvasObject{
		makeScheduleRow(scheduleLabels["shortInterval"], labelWidth, shortInt, valueEntryWidth, labels["shortInterval"]),
		makeScheduleRow(scheduleLabels["shortDuration"], labelWidth, shortDur, valueEntryWidth, labels["shortDuration"]),
		makeScheduleRow(scheduleLabels["longInterval"], labelWidth, longInt, valueEntryWidth, labels["longInterval"]),
		makeScheduleRow(scheduleLabels["longDuration"], labelWidth, longDur, valueEntryWidth, labels["longDuration"]),
	}
}

func (prefs *Window) refreshScheduleLayoutIfNeeded() {
	lang := i18n.NormalizeLanguage(prefs.uiLocalizer.Language())
	if lang == prefs.scheduleLayoutLang {
		return
	}
	labelWidth := scheduleLabelWidthForLang(lang)
	prefs.scheduleSection.Objects = buildScheduleRowGroup(
		prefs.scheduleLabels,
		prefs.labels,
		labelWidth,
		prefs.shortInt,
		prefs.shortDur,
		prefs.longInt,
		prefs.longDur,
	)
	prefs.scheduleSection.Refresh()
	prefs.scheduleLayoutLang = lang
}
