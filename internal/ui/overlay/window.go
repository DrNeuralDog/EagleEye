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

// Config defines overlay visuals.
type Config struct {
	Opacity    uint8
	Fullscreen bool
	Message    string
}

// Session defines a single overlay session.
type Session struct {
	Remaining  time.Duration
	StrictMode bool
	Exercise   animation.ExerciseType
}

// Window manages the overlay UI.
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

// New creates a new overlay window.
func New(ctx context.Context, app fyne.App, config Config, engine *animation.Engine, localizer *i18n.Localizer) *Window {
	if ctx == nil {
		ctx = context.Background()
	}
	if localizer == nil {
		localizer = i18n.New(i18n.LanguageEN)
	}
	window := app.NewWindow("EagleEye")
	if driver, ok := app.Driver().(splashWindowDriver); ok {
		// Splash window is undecorated (no native frame/buttons).
		window = driver.CreateSplashWindow()
	}
	if app.Icon() != nil {
		window.SetIcon(app.Icon())
	}
	window.SetPadded(false)

	fullscreenBackground := canvas.NewRectangle(overlayBackgroundColor(config.Opacity))
	cardBackground := canvas.NewRectangle(overlayBackgroundColor(config.Opacity))
	cardBackground.CornerRadius = overlayCardCornerRadius

	image := canvas.NewImageFromResource(nil)
	image.FillMode = canvas.ImageFillContain

	timerLabel := canvas.NewText("--:--", color.NRGBA{R: 232, G: 190, B: 66, A: 255})
	timerLabel.Alignment = fyne.TextAlignLeading
	timerLabel.TextStyle = fyne.TextStyle{Bold: true}
	timerLabel.TextSize = 16

	titleLabel := canvas.NewText(localizer.T("overlay.title"), color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	titleLabel.Alignment = fyne.TextAlignLeading
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	titleLabel.TextSize = 21

	subtitleLabel := canvas.NewText(config.Message, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	subtitleLabel.Alignment = fyne.TextAlignLeading
	subtitleLabel.TextStyle = fyne.TextStyle{Bold: true}
	subtitleLabel.TextSize = 14

	exerciseLabel := canvas.NewText("", color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	exerciseLabel.Alignment = fyne.TextAlignLeading
	exerciseLabel.TextSize = 17

	skipButton := widget.NewButton(localizer.T("overlay.skip"), nil)

	leftContent := container.New(&leftPanelLayout{}, titleLabel, subtitleLabel, exerciseLabel, timerLabel)
	rightLayout := &rightPanelLayout{}
	rightContent := container.New(rightLayout, image, skipButton)
	content := container.NewGridWithColumns(2, leftContent, rightContent)
	card := container.NewStack(cardBackground, content)
	cardHostLayout := &overlayCardHostLayout{fullscreen: config.Fullscreen}
	cardHost := container.New(cardHostLayout, card)
	root := container.NewStack(fullscreenBackground, cardHost)

	window.SetContent(root)
	overlay := &Window{
		app:              app,
		window:           window,
		rootCtx:          ctx,
		config:           config,
		image:            image,
		timerLabel:       timerLabel,
		skipButton:       skipButton,
		rightPanel:       rightContent,
		rightPanelLayout: rightLayout,
		titleLabel:       titleLabel,
		subtitleLabel:    subtitleLabel,
		exerciseLabel:    exerciseLabel,
		fullscreenBG:     fullscreenBackground,
		cardBackground:   cardBackground,
		cardHost:         cardHost,
		cardHostLayout:   cardHostLayout,
		engine:           engine,
		localizer:        localizer,
		currentExercise:  animation.ExerciseLeftRight,
	}

	overlay.setExerciseUnsafe(animation.ExerciseLeftRight)
	overlay.applyWindowMode()
	overlay.applyNativeOpacity(config.Opacity)

	window.SetCloseIntercept(func() {
		if overlay.onSkip != nil {
			overlay.onSkip()
		}
	})

	return overlay
}

// SetEngine attaches the animation engine.
func (overlay *Window) SetEngine(engine *animation.Engine) {
	overlay.engine = engine
}

// Show starts a new overlay session.
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

// ShowIdle starts the idle animation (long breaks).
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

// Hide closes the overlay and stops animations.
func (overlay *Window) Hide() {
	overlay.releaseClipCursor()
	overlay.stopEngine()
	if overlay.config.Fullscreen {
		overlay.window.SetFullScreen(false)
	}
	overlay.applyNativeTopmost(false)
	overlay.window.Hide()
}

// SetRemaining updates the timer label and keeps the overlay above regular
// windows. Strict mode also re-claims focus so it stays visible across screens.
func (overlay *Window) SetRemaining(remaining time.Duration) {
	overlay.setRemaining(remaining)
	if overlay.strictMode {
		overlay.forceForeground()
		return
	}
	overlay.keepTopmost()
}

// SetStrictMode toggles skip visibility.
func (overlay *Window) SetStrictMode(enabled bool) {
	overlay.setStrictMode(enabled)
}

// SetExercise updates the movement text.
func (overlay *Window) SetExercise(exercise animation.ExerciseType) {
	fyne.Do(func() {
		overlay.setExerciseUnsafe(exercise)
	})
}

// SetOnSkip sets skip handler.
func (overlay *Window) SetOnSkip(handler func()) {
	overlay.onSkip = handler
	overlay.skipButton.OnTapped = func() {
		if overlay.onSkip != nil {
			overlay.onSkip()
		}
	}
}

// UpdateConfig updates overlay visuals. The window mode (fullscreen vs
// windowed) is stored but only applied on the next Show/ShowIdle call,
// so that updating settings while the overlay is hidden does not
// accidentally make it visible.
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

// RefreshLocalization refreshes language-dependent overlay texts.
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

// SetSprite updates the center sprite image.
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

func (overlay *Window) setRemaining(remaining time.Duration) {
	overlay.setRemainingUnsafe(remaining)
}

func (overlay *Window) setStrictMode(enabled bool) {
	overlay.setStrictModeUnsafe(enabled)
}

func (overlay *Window) setRemainingUnsafe(remaining time.Duration) {
	overlay.timerLabel.Text = formatDuration(remaining)
	overlay.timerLabel.Refresh()
}

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

func (overlay *Window) setExerciseUnsafe(exercise animation.ExerciseType) {
	overlay.currentExercise = exercise
	overlay.exerciseLabel.Text = exerciseDescription(exercise, overlay.localizer)
	overlay.exerciseLabel.Refresh()
}

func (overlay *Window) stopEngine() {
	if overlay.cancelCtx != nil {
		overlay.cancelCtx()
		overlay.cancelCtx = nil
	}
}

func (overlay *Window) applyWindowMode() {
	if overlay.cardHostLayout != nil {
		overlay.cardHostLayout.SetFullscreen(overlay.config.Fullscreen)
	}
	if overlay.cardHost != nil {
		overlay.cardHost.Refresh()
	}
	// In windowed mode the fullscreen backdrop shares the card's bounds,
	// so it must be rounded too — otherwise it pokes through the card's
	// rounded corners. In fullscreen it covers the whole screen, so keep
	// it square.
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

func formatDuration(value time.Duration) string {
	if value < 0 {
		value = 0
	}
	seconds := int(value.Seconds())
	minutes := seconds / 60
	seconds = seconds % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func overlayBackgroundColor(alpha uint8) color.NRGBA {
	return color.NRGBA{R: 0, G: 0, B: 0, A: alpha}
}

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
