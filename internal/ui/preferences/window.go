package preferences

import (
	"eagleeye/internal/ui/i18n"
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	prefsWindowWidth          = float32(560)
	languageSelectWrapWidth   = float32(142)
	strictModeTopSpacerHeight = float32(8)
	saveCancelButtonWidth     = float32(130 * 1.4)
	saveCancelButtonHeight    = float32(40)
	// Vertical gap between run-on-startup, language row, and overlay opacity label.
	preferencesMidFormGap = float32(12)
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
	window      fyne.Window
	settings    Settings
	callbacks   Callbacks
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

	scheduleSection    *fyne.Container
	scheduleLayoutLang string

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
	idleTrackingRow := newDelayedHoverInfoRow(idleCheck, window.Canvas(), func() string {
		return uiLocalizer.T("prefs.idleTrackingHelp")
	}, func() {
		idleCheck.SetChecked(!idleCheck.Checked)
	})

	opacity := widget.NewSlider(0.7, 0.95)
	opacity.Value = settings.OverlayOpacity
	opacity.Step = 0.01

	fullscreen := widget.NewCheck("", nil)
	fullscreen.SetChecked(settings.Fullscreen)

	runOnStartup := widget.NewCheck("", nil)
	runOnStartup.SetChecked(settings.RunOnStartup)

	languageSelect := widget.NewSelect(i18n.LanguageOptions(), nil)
	languageSelect.SetSelected(i18n.LanguageDisplayName(settings.Language))

	statusBar, statusDot, statusBarMain, statusBarTimer := newStatusBar()

	heading := canvas.NewText("", theme.Color(theme.ColorNameForeground))
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
		idleTrackingRow,
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

func newVerticalSpacer(height float32) fyne.CanvasObject {
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(1, height))
	return spacer
}
