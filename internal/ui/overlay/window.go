package overlay

import (
	"context"
	"fmt"
	"image/color"
	"time"

	"eagleeye/internal/ui/animation"
	"eagleeye/internal/ui/i18n"

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
	app             fyne.App
	window          fyne.Window
	config          Config
	image           *canvas.Image
	timerLabel      *canvas.Text
	skipButton      *widget.Button
	rightPanel      *fyne.Container
	titleLabel      *canvas.Text
	subtitleLabel   *canvas.Text
	exerciseLabel   *canvas.Text
	fullscreenBG    *canvas.Rectangle
	cardBackground  *canvas.Rectangle
	cardHost        *fyne.Container
	cardHostLayout  *overlayCardHostLayout
	engine          *animation.Engine
	cancelCtx       context.CancelFunc
	onSkip          func()
	localizer       *i18n.Localizer
	currentExercise animation.ExerciseType
}

const (
	overlayWidthFraction  = float32(0.14)
	overlayHeightFraction = float32(0.18)
	defaultScreenWidth    = float32(1920)
	defaultScreenHeight   = float32(1080)
)

type splashWindowDriver interface {
	CreateSplashWindow() fyne.Window
}

// New creates a new overlay window.
func New(app fyne.App, config Config, engine *animation.Engine, localizer *i18n.Localizer) *Window {
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
	rightContent := container.New(&rightPanelLayout{}, image, skipButton)
	content := container.NewGridWithColumns(2, leftContent, rightContent)
	card := container.NewMax(cardBackground, content)
	cardHostLayout := &overlayCardHostLayout{fullscreen: config.Fullscreen}
	cardHost := container.New(cardHostLayout, card)
	root := container.NewMax(fullscreenBackground, cardHost)

	window.SetContent(root)
	overlay := &Window{
		app:             app,
		window:          window,
		config:          config,
		image:           image,
		timerLabel:      timerLabel,
		skipButton:      skipButton,
		rightPanel:      rightContent,
		titleLabel:      titleLabel,
		subtitleLabel:   subtitleLabel,
		exerciseLabel:   exerciseLabel,
		fullscreenBG:    fullscreenBackground,
		cardBackground:  cardBackground,
		cardHost:        cardHost,
		cardHostLayout:  cardHostLayout,
		engine:          engine,
		localizer:       localizer,
		currentExercise: animation.ExerciseLeftRight,
	}

	overlay.setExerciseUnsafe(animation.ExerciseLeftRight)
	overlay.applyWindowMode()
	overlay.applyNativeOpacity(config.Opacity)

	return overlay
}

// SetEngine attaches the animation engine.
func (overlay *Window) SetEngine(engine *animation.Engine) {
	overlay.engine = engine
}

// Show starts a new overlay session.
func (overlay *Window) Show(session Session, spec animation.ExerciseSpec) {
	overlay.stopEngine()
	ctx, cancel := context.WithCancel(context.Background())
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

	if overlay.engine != nil {
		overlay.engine.StartExercise(ctx, spec)
	}
}

// ShowIdle starts the idle animation (long breaks).
func (overlay *Window) ShowIdle(remaining time.Duration, strict bool, idle animation.IdleSpec) {
	overlay.stopEngine()
	ctx, cancel := context.WithCancel(context.Background())
	overlay.cancelCtx = cancel

	overlay.setRemainingUnsafe(remaining)
	overlay.setExerciseUnsafe(animation.ExerciseBlink)
	overlay.setStrictModeUnsafe(strict)
	overlay.applyWindowMode()
	overlay.window.Show()
	overlay.applyNativeTopmost(true)
	overlay.applyNativeOpacity(overlay.config.Opacity)
	overlay.window.RequestFocus()

	if overlay.engine != nil {
		overlay.engine.StartIdle(ctx, idle)
	}
}

// Hide closes the overlay and stops animations.
func (overlay *Window) Hide() {
	overlay.stopEngine()
	if overlay.config.Fullscreen {
		overlay.window.SetFullScreen(false)
	}
	overlay.applyNativeTopmost(false)
	overlay.window.Hide()
}

// SetRemaining updates the timer label.
func (overlay *Window) SetRemaining(remaining time.Duration) {
	overlay.setRemaining(remaining)
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

// UpdateConfig updates overlay visuals.
func (overlay *Window) UpdateConfig(config Config) {
	overlay.config = config
	overlay.subtitleLabel.Text = config.Message
	updatedColor := overlayBackgroundColor(config.Opacity)
	overlay.fullscreenBG.FillColor = updatedColor
	overlay.cardBackground.FillColor = updatedColor
	overlay.applyNativeOpacity(config.Opacity)
	overlay.applyWindowMode()
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
		overlay.image.Resource = resource
		overlay.image.Refresh()
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
	if overlay.config.Fullscreen {
		overlay.window.SetFullScreen(true)
		return
	}
	overlay.window.SetFullScreen(false)
	overlay.resizeToScreenFraction()
}

func (overlay *Window) resizeToScreenFraction() {
	screenSize := overlay.resolveScreenSize()
	overlaySize := calculateOverlayCardSize(screenSize, overlay.window.Content().MinSize())
	overlay.window.Resize(overlaySize)
	overlay.window.CenterOnScreen()
}

func (overlay *Window) resolveScreenSize() fyne.Size {
	screenSize := fyne.NewSize(defaultScreenWidth, defaultScreenHeight)
	canvasSize := overlay.window.Canvas().Size()
	// Canvas size can be reused as a proxy for monitor size when it is clearly screen-like.
	if canvasSize.Width >= 1024 && canvasSize.Height >= 720 {
		screenSize = canvasSize
	}
	return screenSize
}

func calculateOverlayCardSize(screenSize fyne.Size, minSize fyne.Size) fyne.Size {
	if screenSize.Width <= 0 {
		screenSize.Width = defaultScreenWidth
	}
	if screenSize.Height <= 0 {
		screenSize.Height = defaultScreenHeight
	}

	width := screenSize.Width * overlayWidthFraction
	height := screenSize.Height * overlayHeightFraction
	if width < minSize.Width {
		width = minSize.Width
	}
	if height < minSize.Height {
		height = minSize.Height
	}
	if width > screenSize.Width {
		width = screenSize.Width
	}
	if height > screenSize.Height {
		height = screenSize.Height
	}
	return fyne.NewSize(width, height)
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

type rightPanelLayout struct{}

func (layout *rightPanelLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 2 {
		return
	}
	image := objects[0]
	skip := objects[1]

	skipSize := skip.MinSize()
	if !skip.Visible() {
		side := size.Height
		if side > size.Width {
			side = size.Width
		}
		x := size.Width - side
		if x < 0 {
			x = 0
		}
		y := (size.Height - side) / 2
		if y < 0 {
			y = 0
		}
		image.Move(fyne.NewPos(x, y))
		image.Resize(fyne.NewSize(side, side))
		return
	}

	skipHeight := skipSize.Height
	if skipHeight > size.Height*0.25 {
		skipHeight = size.Height * 0.25
	}
	imageAreaHeight := size.Height - skipHeight
	if imageAreaHeight < 0 {
		imageAreaHeight = 0
	}

	margin := imageAreaHeight * 0.05
	side := imageAreaHeight * 0.90
	if side > size.Width-margin {
		side = size.Width - margin
	}
	if side < 0 {
		side = 0
	}
	x := size.Width - margin - side
	if x < 0 {
		x = 0
	}
	y := margin
	image.Move(fyne.NewPos(x, y))
	image.Resize(fyne.NewSize(side, side))

	skipWidth := skipSize.Width
	skipWidth = skipWidth * 1.4
	if skipWidth > size.Width {
		skipWidth = size.Width
	}
	skipX := x + side - skipWidth
	if skipX < 0 {
		skipX = 0
	}
	skipY := imageAreaHeight + (skipHeight-skipSize.Height)/2
	if skipY < 0 {
		skipY = 0
	}
	skip.Move(fyne.NewPos(skipX, skipY))
	skip.Resize(fyne.NewSize(skipWidth, skipSize.Height))
}

func (layout *rightPanelLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) < 2 {
		return fyne.NewSize(0, 0)
	}
	imageMin := objects[0].MinSize()
	skipMin := objects[1].MinSize()
	if !objects[1].Visible() {
		return imageMin
	}
	width := imageMin.Width
	if skipMin.Width > width {
		width = skipMin.Width
	}
	return fyne.NewSize(width, imageMin.Height+skipMin.Height)
}

type leftPanelLayout struct{}

func (layout *leftPanelLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 4 {
		return
	}
	title := objects[0]
	subtitle := objects[1]
	exercise := objects[2]
	timer := objects[3]

	pad := size.Height * 0.05
	availableWidth := size.Width - pad*2
	if availableWidth < 0 {
		availableWidth = 0
	}

	titleSize := title.MinSize()
	title.Move(fyne.NewPos(pad, pad))
	title.Resize(fyne.NewSize(availableWidth, titleSize.Height))

	subtitleSize := subtitle.MinSize()
	subtitleY := pad + titleSize.Height + 6
	subtitle.Move(fyne.NewPos(pad, subtitleY))
	subtitle.Resize(fyne.NewSize(availableWidth, subtitleSize.Height))

	exerciseSize := exercise.MinSize()
	exerciseY := subtitleY + subtitleSize.Height + 8
	exercise.Move(fyne.NewPos(pad, exerciseY))
	exercise.Resize(fyne.NewSize(availableWidth, exerciseSize.Height))

	timerSize := timer.MinSize()
	timerY := size.Height - pad - timerSize.Height
	if timerY < 0 {
		timerY = 0
	}
	timer.Move(fyne.NewPos(pad, timerY))
	timer.Resize(timerSize)
}

func (layout *leftPanelLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) < 4 {
		return fyne.NewSize(0, 0)
	}
	titleSize := objects[0].MinSize()
	subtitleSize := objects[1].MinSize()
	exerciseSize := objects[2].MinSize()
	timerSize := objects[3].MinSize()

	width := titleSize.Width
	if subtitleSize.Width > width {
		width = subtitleSize.Width
	}
	if exerciseSize.Width > width {
		width = exerciseSize.Width
	}
	if timerSize.Width > width {
		width = timerSize.Width
	}
	height := titleSize.Height + subtitleSize.Height + exerciseSize.Height + timerSize.Height + 40
	return fyne.NewSize(width+20, height)
}

type overlayCardHostLayout struct {
	fullscreen bool
}

func (layout *overlayCardHostLayout) SetFullscreen(fullscreen bool) {
	layout.fullscreen = fullscreen
}

func (layout *overlayCardHostLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	card := objects[0]
	if !layout.fullscreen {
		card.Move(fyne.NewPos(0, 0))
		card.Resize(size)
		return
	}

	cardSize := calculateOverlayCardSize(size, card.MinSize())
	x := (size.Width - cardSize.Width) / 2
	if x < 0 {
		x = 0
	}
	y := (size.Height - cardSize.Height) / 2
	if y < 0 {
		y = 0
	}
	card.Move(fyne.NewPos(x, y))
	card.Resize(cardSize)
}

func (layout *overlayCardHostLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.NewSize(0, 0)
	}
	return objects[0].MinSize()
}
