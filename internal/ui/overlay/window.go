package overlay

import (
	"context"
	"eagleeye/internal/ui/animation"
	"eagleeye/internal/ui/i18n"
	"fmt"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Config defines overlay visuals
type Config struct {
	Opacity    uint8
	Fullscreen bool
	Message    string
}

// Session defines a single overlay session
type Session struct {
	Remaining  time.Duration
	StrictMode bool
	Exercise   animation.ExerciseType
}

// Window manages the overlay UI
type Window struct {
	app              fyne.App
	window           fyne.Window
	rootCtx          context.Context
	config           Config
	image            *canvas.Image
	timerLabel       *canvas.Text
	skipButton       *widget.Button
	rightPanel       *fyne.Container
	rightPanelLayout *rightPanelLayout
	titleLabel       *canvas.Text
	subtitleLabel    *canvas.Text
	exerciseLabel    *canvas.Text
	fullscreenBG     *canvas.Rectangle
	cardBackground   *canvas.Rectangle
	cardHost         *fyne.Container
	cardHostLayout   *overlayCardHostLayout
	engine           *animation.Engine
	cancelCtx        context.CancelFunc
	onSkip           func()
	localizer        *i18n.Localizer
	currentExercise  animation.ExerciseType
	strictMode       bool
	cachedHWND       uintptr
}

type splashWindowDriver interface {
	CreateSplashWindow() fyne.Window
}

type overlayLabels struct {
	timer    *canvas.Text
	title    *canvas.Text
	subtitle *canvas.Text
	exercise *canvas.Text
}

type overlayView struct {
	root fyne.CanvasObject

	image      *canvas.Image
	skipButton *widget.Button

	rightPanel       *fyne.Container
	rightPanelLayout *rightPanelLayout

	labels overlayLabels

	fullscreenBG   *canvas.Rectangle
	cardBackground *canvas.Rectangle
	cardHost       *fyne.Container
	cardHostLayout *overlayCardHostLayout
}

// New creates a new overlay window
func New(ctx context.Context, app fyne.App, config Config, engine *animation.Engine, localizer *i18n.Localizer) *Window {
	ctx = defaultOverlayContext(ctx)
	localizer = defaultOverlayLocalizer(localizer)
	window := newOverlayWindow(app)
	view := newOverlayView(config, localizer)

	window.SetContent(view.root)

	overlay := newWindowState(ctx, app, window, config, engine, localizer, view)
	overlay.initializeWindow()
	overlay.bindCloseHandler()

	return overlay
}

// defaultOverlayContext keeps scheduling safe when callers pass nil
func defaultOverlayContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}

	return ctx
}

// defaultOverlayLocalizer keeps overlay text available without caller setup
func defaultOverlayLocalizer(localizer *i18n.Localizer) *i18n.Localizer {
	if localizer == nil {
		return i18n.New(i18n.LanguageEN)
	}

	return localizer
}

// newOverlayWindow creates the native Fyne window shell
func newOverlayWindow(app fyne.App) fyne.Window {
	window := app.NewWindow("EagleEye")

	if driver, ok := app.Driver().(splashWindowDriver); ok {
		window = driver.CreateSplashWindow()
	}

	if app.Icon() != nil {
		window.SetIcon(app.Icon())
	}

	window.SetPadded(false)

	return window
}

// newOverlayView builds the canvas tree used by every overlay session
func newOverlayView(config Config, localizer *i18n.Localizer) *overlayView {
	fullscreenBackground := canvas.NewRectangle(overlayBackgroundColor(config.Opacity))
	cardBackground := canvas.NewRectangle(overlayBackgroundColor(config.Opacity))
	cardBackground.CornerRadius = overlayCardCornerRadius

	image := canvas.NewImageFromResource(nil)
	image.FillMode = canvas.ImageFillContain

	labels := newOverlayLabels(config, localizer)
	skipButton := widget.NewButton(localizer.T("overlay.skip"), nil)

	leftContent := container.New(&leftPanelLayout{}, labels.title, labels.subtitle, labels.exercise, labels.timer)

	rightLayout := &rightPanelLayout{}
	rightContent := container.New(rightLayout, image, skipButton)

	content := container.NewGridWithColumns(2, leftContent, rightContent)
	card := container.NewStack(cardBackground, content)

	cardHostLayout := &overlayCardHostLayout{fullscreen: config.Fullscreen}
	cardHost := container.New(cardHostLayout, card)
	root := container.NewStack(fullscreenBackground, cardHost)

	return &overlayView{
		root:             root,
		image:            image,
		skipButton:       skipButton,
		rightPanel:       rightContent,
		rightPanelLayout: rightLayout,
		labels:           labels,
		fullscreenBG:     fullscreenBackground,
		cardBackground:   cardBackground,
		cardHost:         cardHost,
		cardHostLayout:   cardHostLayout,
	}
}

// newOverlayLabels creates text nodes before localization refreshes them
func newOverlayLabels(config Config, localizer *i18n.Localizer) overlayLabels {
	timer := canvas.NewText("--:--", color.NRGBA{R: 232, G: 190, B: 66, A: 255})
	timer.Alignment = fyne.TextAlignLeading
	timer.TextStyle = fyne.TextStyle{Bold: true}
	timer.TextSize = 16

	title := canvas.NewText(localizer.T("overlay.title"), color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	title.Alignment = fyne.TextAlignLeading
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 21

	subtitle := canvas.NewText(config.Message, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	subtitle.Alignment = fyne.TextAlignLeading
	subtitle.TextStyle = fyne.TextStyle{Bold: true}
	subtitle.TextSize = 14

	exercise := canvas.NewText("", color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	exercise.Alignment = fyne.TextAlignLeading
	exercise.TextSize = 17

	return overlayLabels{
		timer:    timer,
		title:    title,
		subtitle: subtitle,
		exercise: exercise,
	}
}

// newWindowState wires view objects into the overlay runtime state
func newWindowState(
	ctx context.Context,
	app fyne.App,
	window fyne.Window,
	config Config,
	engine *animation.Engine,
	localizer *i18n.Localizer,
	view *overlayView,
) *Window {
	return &Window{
		app:     app,
		window:  window,
		rootCtx: ctx,
		config:  config,

		image:            view.image,
		timerLabel:       view.labels.timer,
		skipButton:       view.skipButton,
		rightPanel:       view.rightPanel,
		rightPanelLayout: view.rightPanelLayout,
		titleLabel:       view.labels.title,
		subtitleLabel:    view.labels.subtitle,
		exerciseLabel:    view.labels.exercise,
		fullscreenBG:     view.fullscreenBG,
		cardBackground:   view.cardBackground,
		cardHost:         view.cardHost,
		cardHostLayout:   view.cardHostLayout,

		engine:          engine,
		localizer:       localizer,
		currentExercise: animation.ExerciseLeftRight,
	}
}

// initializeWindow applies default exercise text and native window settings
func (overlay *Window) initializeWindow() {
	overlay.setExerciseUnsafe(animation.ExerciseLeftRight)

	overlay.applyWindowMode()
	overlay.applyNativeOpacity(overlay.config.Opacity)
}

// bindCloseHandler maps native close to the configured skip behavior
func (overlay *Window) bindCloseHandler() {
	overlay.window.SetCloseIntercept(func() {
		if overlay.onSkip != nil {
			overlay.onSkip()
		}
	})
}

// SetEngine attaches the animation engine
func (overlay *Window) SetEngine(engine *animation.Engine) {
	overlay.engine = engine
}

// Show starts a new overlay session
func (overlay *Window) Show(session Session, spec animation.ExerciseSpec) {
	overlay.stopEngine()
	ctx, cancel := context.WithCancel(overlay.rootCtx)
	overlay.cancelCtx = cancel

	if session.Exercise == animation.ExerciseLookOutside && session.Remaining <= 0 {
		session.Exercise = animation.ExerciseLookOutside
	}

	spec.Duration = session.Remaining
	spec.Type = session.Exercise

	overlay.setRemainingUnsafe(session.Remaining)
	overlay.setExerciseUnsafe(session.Exercise)
	overlay.setStrictModeUnsafe(session.StrictMode)
	overlay.applyWindowMode()
	overlay.window.Show()

	overlay.applyNativeTopmost(true)
	overlay.applyNativeOpacity(overlay.config.Opacity)
	overlay.window.RequestFocus()

	overlay.scheduleInitialFocus(ctx)
	overlay.scheduleTopmostKeep(ctx)
	overlay.scheduleNativeShape(ctx, overlay.nativeShapeRadius())

	if overlay.engine != nil {
		overlay.engine.StartExercise(ctx, spec)
	}
}

// ShowIdle starts the idle animation for long breaks
func (overlay *Window) ShowIdle(remaining time.Duration, strict bool, idle animation.IdleSpec) {
	overlay.stopEngine()

	ctx, cancel := context.WithCancel(overlay.rootCtx)
	overlay.cancelCtx = cancel

	overlay.setRemainingUnsafe(remaining)
	overlay.setExerciseUnsafe(animation.ExerciseBlink)
	overlay.setStrictModeUnsafe(strict)
	overlay.applyWindowMode()
	overlay.window.Show()

	overlay.applyNativeTopmost(true)
	overlay.applyNativeOpacity(overlay.config.Opacity)
	overlay.window.RequestFocus()

	overlay.scheduleInitialFocus(ctx)
	overlay.scheduleTopmostKeep(ctx)
	overlay.scheduleNativeShape(ctx, overlay.nativeShapeRadius())

	if overlay.engine != nil {
		overlay.engine.StartIdle(ctx, idle)
	}
}

// Hide closes the overlay and stops animations
func (overlay *Window) Hide() {
	overlay.releaseClipCursor()
	overlay.stopEngine()

	if overlay.config.Fullscreen {
		overlay.window.SetFullScreen(false)
	}

	overlay.applyNativeTopmost(false)
	overlay.window.Hide()
}

// SetRemaining updates the timer and keeps the overlay above regular windows
func (overlay *Window) SetRemaining(remaining time.Duration) {
	overlay.setRemaining(remaining)
	if overlay.strictMode {
		overlay.forceForeground()

		return
	}
	overlay.keepTopmost()
}

// SetStrictMode toggles skip visibility
func (overlay *Window) SetStrictMode(enabled bool) {
	overlay.setStrictMode(enabled)
}

// SetExercise updates the movement text
func (overlay *Window) SetExercise(exercise animation.ExerciseType) {
	fyne.Do(func() {
		overlay.setExerciseUnsafe(exercise)
	})
}

// SetOnSkip sets skip handler
func (overlay *Window) SetOnSkip(handler func()) {
	overlay.onSkip = handler
	overlay.skipButton.OnTapped = func() {
		if overlay.onSkip != nil {
			overlay.onSkip()
		}
	}
}

// UpdateConfig applies visual settings and stores window mode for the next show
func (overlay *Window) UpdateConfig(config Config) {
	overlay.config = config
	overlay.subtitleLabel.Text = config.Message
	updatedColor := overlayBackgroundColor(config.Opacity)
	overlay.fullscreenBG.FillColor = updatedColor
	overlay.cardBackground.FillColor = updatedColor

	canvas.Refresh(overlay.fullscreenBG)
	canvas.Refresh(overlay.cardBackground)
	overlay.titleLabel.Refresh()
	overlay.subtitleLabel.Refresh()
	overlay.exerciseLabel.Refresh()
}

// RefreshLocalization refreshes language-dependent overlay texts
func (overlay *Window) RefreshLocalization() {
	fyne.Do(func() {
		overlay.titleLabel.Text = overlay.localizer.T("overlay.title")
		overlay.subtitleLabel.Text = overlay.config.Message

		overlay.skipButton.SetText(overlay.localizer.T("overlay.skip"))
		overlay.setExerciseUnsafe(overlay.currentExercise)

		overlay.titleLabel.Refresh()
		overlay.subtitleLabel.Refresh()
		overlay.exerciseLabel.Refresh()
	})
}

// SetSprite updates the center sprite image
func (overlay *Window) SetSprite(resource fyne.Resource) {
	fyne.Do(func() {
		transform := spriteTransformForResource(resource)
		overlay.image.Resource = resource

		if transform.stretch {
			overlay.image.FillMode = canvas.ImageFillStretch
		} else {
			overlay.image.FillMode = canvas.ImageFillContain
		}

		if overlay.rightPanelLayout != nil {
			overlay.rightPanelLayout.SetSpriteTransform(transform)
		}

		overlay.image.Refresh()

		if overlay.rightPanel != nil {
			overlay.rightPanel.Refresh()
		}
	})
}

// setRemaining updates the timer from synchronous overlay paths
func (overlay *Window) setRemaining(remaining time.Duration) {
	overlay.setRemainingUnsafe(remaining)
}

// setStrictMode updates strict mode from synchronous overlay paths
func (overlay *Window) setStrictMode(enabled bool) {
	overlay.setStrictModeUnsafe(enabled)
}

func (overlay *Window) setRemainingUnsafe(remaining time.Duration) {
	overlay.timerLabel.Text = formatDuration(remaining)
	overlay.timerLabel.Refresh()
}

// setStrictModeUnsafe updates skip button state on the current UI context
func (overlay *Window) setStrictModeUnsafe(enabled bool) {
	overlay.strictMode = enabled
	if enabled {
		overlay.skipButton.Hide()
		overlay.skipButton.Disable()

		if overlay.rightPanel != nil {
			overlay.rightPanel.Refresh()
		}
		return
	}

	overlay.skipButton.Show()
	overlay.skipButton.Enable()

	if overlay.rightPanel != nil {
		overlay.rightPanel.Refresh()
	}
}

// setExerciseUnsafe updates exercise text on the current UI context
func (overlay *Window) setExerciseUnsafe(exercise animation.ExerciseType) {
	overlay.currentExercise = exercise
	overlay.exerciseLabel.Text = exerciseDescription(exercise, overlay.localizer)

	overlay.exerciseLabel.Refresh()
}

// stopEngine cancels the active animation session
func (overlay *Window) stopEngine() {
	if overlay.cancelCtx != nil {
		overlay.cancelCtx()
		overlay.cancelCtx = nil
	}
}

// applyWindowMode syncs fullscreen/windowed layout with the stored config
func (overlay *Window) applyWindowMode() {
	if overlay.cardHostLayout != nil {
		overlay.cardHostLayout.SetFullscreen(overlay.config.Fullscreen)
	}

	if overlay.cardHost != nil {
		overlay.cardHost.Refresh()
	}

	if overlay.config.Fullscreen {
		overlay.fullscreenBG.CornerRadius = 0
	} else {
		overlay.fullscreenBG.CornerRadius = overlayCardCornerRadius
	}

	canvas.Refresh(overlay.fullscreenBG)

	if overlay.config.Fullscreen {
		overlay.window.SetFullScreen(true)
		return
	}

	overlay.window.SetFullScreen(false)
	overlay.resizeToScreenFraction()
}

// formatDuration renders a non-negative mm:ss timer string
func formatDuration(value time.Duration) string {
	if value < 0 {
		value = 0
	}

	seconds := int(value.Seconds())
	minutes := seconds / 60
	seconds = seconds % 60

	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

// overlayBackgroundColor converts opacity into the overlay background color
func overlayBackgroundColor(alpha uint8) color.NRGBA {
	return color.NRGBA{R: 0, G: 0, B: 0, A: alpha}
}

// exerciseDescription resolves the localized instruction for an exercise
func exerciseDescription(exercise animation.ExerciseType, localizer *i18n.Localizer) string {
	switch exercise {
	case animation.ExerciseLeftRight:
		return localizer.T("overlay.exercise.leftRight")
	case animation.ExerciseUpDown:
		return localizer.T("overlay.exercise.upDown")
	case animation.ExerciseBlink:
		return localizer.T("overlay.exercise.blink")
	case animation.ExerciseLookOutside:
		return localizer.T("overlay.exercise.lookOut")
	default:
		return ""
	}
}
