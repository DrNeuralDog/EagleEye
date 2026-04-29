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
	preferencesMidFormGap     = float32(12)
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

	labels         map[string]*widget.Label
	scheduleLabels map[string]*widget.Label

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

type scheduleEntries struct {
	shortInt *widget.Entry
	shortDur *widget.Entry
	longInt  *widget.Entry
	longDur  *widget.Entry
}

type preferenceChecks struct {
	strict          *widget.Check
	idleCheck       *widget.Check
	idleTrackingRow fyne.CanvasObject
	fullscreen      *widget.Check
	runOnStartup    *widget.Check
}

type languageControls struct {
	label     *widget.Label
	selectBox *widget.Select
	row       fyne.CanvasObject
}

type footerControls struct {
	saveButton        *widget.Button
	cancelButton      *widget.Button
	timerToggleButton *widget.Button
	content           fyne.CanvasObject
}

type preferencesView struct {
	content fyne.CanvasObject

	labels         map[string]*widget.Label
	scheduleLabels map[string]*widget.Label

	heading            *canvas.Text
	entries            scheduleEntries
	checks             preferenceChecks
	opacity            *widget.Slider
	language           languageControls
	overlayOpacityText *widget.Label
	footer             footerControls

	statusIndicatorDot *canvas.Circle
	statusBarMain      *canvas.Text
	statusBarTimer     *canvas.Text

	scheduleSection    *fyne.Container
	scheduleLayoutLang string
}

// New creates a preferences window.
func New(app fyne.App, settings Settings, callbacks Callbacks) *Window {
	uiLocalizer := i18n.New(settings.Language)
	window := newPreferencesWindow(app, uiLocalizer)
	view := newPreferencesView(window, settings, uiLocalizer)

	window.SetContent(view.content)
	configurePreferencesWindow(window)

	prefs := newWindowState(window, settings, callbacks, uiLocalizer, view)
	prefs.bindActions()

	prefs.RefreshLocalization()
	prefs.SetServiceNotStarted()

	return prefs
}

func newPreferencesWindow(app fyne.App, localizer *i18n.Localizer) fyne.Window {
	window := app.NewWindow(localizer.T("prefs.windowTitle"))

	if app.Icon() != nil {
		window.SetIcon(app.Icon())
	}

	return window
}

func configurePreferencesWindow(window fyne.Window) {
	window.Resize(fyne.NewSize(prefsWindowWidth, 520))
	window.SetFixedSize(true)
}

func newPreferencesView(window fyne.Window, settings Settings, localizer *i18n.Localizer) *preferencesView {
	entries := newScheduleEntries(settings)
	labels, scheduleLabels := newScheduleLabels()
	scheduleSection, scheduleLayoutLang := newScheduleSection(labels, scheduleLabels, entries, settings.Language)

	checks := newPreferenceChecks(window, settings, localizer)
	language := newLanguageControls(settings)
	opacity, overlayOpacityLabel := newOpacityControls(settings)
	footer := newFooterControls()
	statusBar, statusDot, statusBarMain, statusBarTimer := newStatusBar()

	heading := newPreferencesHeading()
	form := newPreferencesForm(heading, scheduleSection, checks, language.row, overlayOpacityLabel, opacity)
	content := newPreferencesContent(form, footer.content, statusBar)

	return &preferencesView{
		content:            content,
		labels:             labels,
		scheduleLabels:     scheduleLabels,
		heading:            heading,
		entries:            entries,
		checks:             checks,
		opacity:            opacity,
		language:           language,
		overlayOpacityText: overlayOpacityLabel,
		footer:             footer,
		statusIndicatorDot: statusDot,
		statusBarMain:      statusBarMain,
		statusBarTimer:     statusBarTimer,
		scheduleSection:    scheduleSection,
		scheduleLayoutLang: scheduleLayoutLang,
	}
}

func newScheduleEntries(settings Settings) scheduleEntries {
	return scheduleEntries{
		shortInt: newNumberEntry(int(settings.ShortInterval.Minutes())),
		shortDur: newNumberEntry(int(settings.ShortDuration.Seconds())),
		longInt:  newNumberEntry(int(settings.LongInterval.Minutes())),
		longDur:  newNumberEntry(int(settings.LongDuration.Minutes())),
	}
}

func newNumberEntry(value int) *widget.Entry {
	entry := widget.NewEntry()
	entry.SetText(fmt.Sprintf("%d", value))

	return entry
}

func newScheduleLabels() (map[string]*widget.Label, map[string]*widget.Label) {
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

	return labels, scheduleLabels
}

func newScheduleSection(labels map[string]*widget.Label, scheduleLabels map[string]*widget.Label, entries scheduleEntries, language string) (*fyne.Container, string) {
	layoutLang := i18n.NormalizeLanguage(language)
	labelWidth := scheduleLabelWidthForLang(layoutLang)
	scheduleRows := buildScheduleRowGroup(
		scheduleLabels,
		labels,
		labelWidth,
		entries.shortInt,
		entries.shortDur,
		entries.longInt,
		entries.longDur,
	)

	return container.NewVBox(scheduleRows...), layoutLang
}

func newPreferenceChecks(window fyne.Window, settings Settings, localizer *i18n.Localizer) preferenceChecks {
	strict := widget.NewCheck("", nil)
	strict.SetChecked(settings.StrictMode)

	idleCheck := widget.NewCheck("", nil)
	idleCheck.SetChecked(settings.IdleEnabled)
	idleTrackingRow := newDelayedHoverInfoRow(idleCheck, window.Canvas(), func() string {
		return localizer.T("prefs.idleTrackingHelp")
	}, func() {
		idleCheck.SetChecked(!idleCheck.Checked)
	})

	fullscreen := widget.NewCheck("", nil)
	fullscreen.SetChecked(settings.Fullscreen)

	runOnStartup := widget.NewCheck("", nil)
	runOnStartup.SetChecked(settings.RunOnStartup)

	return preferenceChecks{
		strict:          strict,
		idleCheck:       idleCheck,
		idleTrackingRow: idleTrackingRow,
		fullscreen:      fullscreen,
		runOnStartup:    runOnStartup,
	}
}

func newLanguageControls(settings Settings) languageControls {
	label := widget.NewLabel("")
	selectBox := widget.NewSelect(i18n.LanguageOptions(), nil)
	selectBox.SetSelected(i18n.LanguageDisplayName(settings.Language))

	selectWrap := container.NewGridWrap(
		fyne.NewSize(languageSelectWrapWidth, selectBox.MinSize().Height),
		selectBox,
	)
	row := container.NewHBox(label, selectWrap, layout.NewSpacer())

	return languageControls{
		label:     label,
		selectBox: selectBox,
		row:       row,
	}
}

func newOpacityControls(settings Settings) (*widget.Slider, *widget.Label) {
	opacity := widget.NewSlider(0.7, 0.95)
	opacity.Value = settings.OverlayOpacity
	opacity.Step = 0.01

	return opacity, widget.NewLabel("")
}

func newFooterControls() footerControls {
	saveButton := widget.NewButton("", nil)
	cancelButton := widget.NewButton("", nil)
	timerToggleButton := widget.NewButton("", nil)
	timerToggleButton.Disable()

	saveWrap := container.NewGridWrap(fyne.NewSize(saveCancelButtonWidth, saveCancelButtonHeight), saveButton)
	cancelWrap := container.NewGridWrap(fyne.NewSize(saveCancelButtonWidth, saveCancelButtonHeight), cancelButton)
	buttons := container.NewHBox(saveWrap, layout.NewSpacer(), cancelWrap)
	content := container.NewVBox(newVerticalSpacer(15), buttons, timerToggleButton)

	return footerControls{
		saveButton:        saveButton,
		cancelButton:      cancelButton,
		timerToggleButton: timerToggleButton,
		content:           content,
	}
}

func newPreferencesHeading() *canvas.Text {
	heading := canvas.NewText("", theme.Color(theme.ColorNameForeground))
	heading.TextSize = 19
	heading.TextStyle = fyne.TextStyle{Bold: true}
	heading.Alignment = fyne.TextAlignCenter

	return heading
}

func newPreferencesForm(
	heading *canvas.Text,
	scheduleSection fyne.CanvasObject,
	checks preferenceChecks,
	languageRow fyne.CanvasObject,
	overlayOpacityLabel *widget.Label,
	opacity *widget.Slider,
) fyne.CanvasObject {
	return container.NewVBox(
		newVerticalSpacer(5),
		container.NewCenter(heading),
		newVerticalSpacer(20),
		scheduleSection,
		newVerticalSpacer(strictModeTopSpacerHeight),
		checks.strict,
		checks.idleTrackingRow,
		checks.fullscreen,
		checks.runOnStartup,
		newVerticalSpacer(preferencesMidFormGap),
		languageRow,
		newVerticalSpacer(preferencesMidFormGap),
		overlayOpacityLabel,
		opacity,
	)
}

func newPreferencesContent(form fyne.CanvasObject, footer fyne.CanvasObject, statusBar fyne.CanvasObject) fyne.CanvasObject {
	center := container.NewVBox(form, footer, newVerticalSpacer(statusBarFromFooterGap))

	return container.NewBorder(nil, statusBar, nil, nil, center)
}

func newWindowState(window fyne.Window, settings Settings, callbacks Callbacks, localizer *i18n.Localizer, view *preferencesView) *Window {
	return &Window{
		window:              window,
		settings:            settings,
		callbacks:           callbacks,
		uiLocalizer:         localizer,
		labels:              view.labels,
		scheduleLabels:      view.scheduleLabels,
		heading:             view.heading,
		shortInt:            view.entries.shortInt,
		shortDur:            view.entries.shortDur,
		longInt:             view.entries.longInt,
		longDur:             view.entries.longDur,
		strict:              view.checks.strict,
		idleCheck:           view.checks.idleCheck,
		opacity:             view.opacity,
		fullscreen:          view.checks.fullscreen,
		runOnStartup:        view.checks.runOnStartup,
		languageLabel:       view.language.label,
		languageSelect:      view.language.selectBox,
		overlayOpacityText:  view.overlayOpacityText,
		saveButton:          view.footer.saveButton,
		cancelButton:        view.footer.cancelButton,
		statusIndicatorDot:  view.statusIndicatorDot,
		statusBarMain:       view.statusBarMain,
		statusBarTimer:      view.statusBarTimer,
		timerToggleButton:   view.footer.timerToggleButton,
		scheduleSection:     view.scheduleSection,
		scheduleLayoutLang:  view.scheduleLayoutLang,
		currentServiceState: serviceStateNotStarted,
	}
}

func (prefs *Window) bindActions() {
	prefs.saveButton.OnTapped = prefs.handleSave
	prefs.languageSelect.OnChanged = func(_ string) {
		prefs.uiLocalizer.SetLanguage(i18n.LanguageFromDisplayName(prefs.languageSelect.Selected))
		prefs.RefreshLocalization()
	}
	prefs.cancelButton.OnTapped = func() {
		prefs.dismiss(false)
	}
	prefs.timerToggleButton.OnTapped = func() {
		if prefs.callbacks.OnToggleTimer != nil {
			prefs.callbacks.OnToggleTimer()
		}
	}
	prefs.window.SetCloseIntercept(func() {
		prefs.dismiss(false)
	})
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
