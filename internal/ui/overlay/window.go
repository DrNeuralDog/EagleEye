package overlay

import (
	"context"
	"fmt"
	"image/color"
	"strings"
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
	app              fyne.App
	window           fyne.Window
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

const (
	overlayWidthFraction    = float32(0.14)
	overlayHeightFraction   = float32(0.18)
	defaultScreenWidth      = float32(1920)
	defaultScreenHeight     = float32(1080)
	overlayCardCornerRadius = float32(32)
	overlayBottomClearance  = float32(12)
	skipModeImageScale      = float32(1.05)
	oversizedSpriteScale    = float32(0.0)
	oversizedSpriteHeight   = float32(0.0)
	oversizedSpriteWidth    = float32(1.)
	oversizedSpriteOffsetY  = float32(-0.0)
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
	card := container.NewMax(cardBackground, content)
	cardHostLayout := &overlayCardHostLayout{fullscreen: config.Fullscreen}
	cardHost := container.New(cardHostLayout, card)
	root := container.NewMax(fullscreenBackground, cardHost)

	window.SetContent(root)
	overlay := &Window{
		app:              app,
		window:           window,
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
	overlay.scheduleInitialFocus()
	overlay.scheduleTopmostKeep(ctx)
	overlay.scheduleNativeShape(overlay.nativeShapeRadius())

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
	overlay.scheduleInitialFocus()
	overlay.scheduleTopmostKeep(ctx)
	overlay.scheduleNativeShape(overlay.nativeShapeRadius())

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

// scheduleInitialFocus re-applies native attributes shortly after the
// window appears, giving the OS time to finish compositing. This avoids
// the brief transparent flash on first show.
func (overlay *Window) scheduleInitialFocus() {
	go func() {
		for _, delay := range []time.Duration{
			50 * time.Millisecond,
			150 * time.Millisecond,
			300 * time.Millisecond,
		} {
			time.Sleep(delay)
			overlay.forceForeground()
		}
	}()
}

func (overlay *Window) scheduleTopmostKeep(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()

		overlay.keepTopmost()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				overlay.keepTopmost()
			}
		}
	}()
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

// nativeShapeRadius is the rounded-corner radius to apply to the
// native HWND for the current window mode. Fullscreen returns 0 so
// the window covers the whole screen without clipping.
func (overlay *Window) nativeShapeRadius() int32 {
	if overlay.config.Fullscreen {
		return 0
	}
	return int32(overlayCardCornerRadius)
}

// scheduleNativeShape re-applies the rounded native window region a
// few times with small delays after window.Show(). Retries are needed
// because the HWND can take a moment to receive its final client size
// via Windows message pumping, and because Fyne's initial layout may
// trigger an additional resize that clears the region.
func (overlay *Window) scheduleNativeShape(radius int32) {
	go func() {
		for _, delay := range []time.Duration{
			0,
			50 * time.Millisecond,
			150 * time.Millisecond,
			300 * time.Millisecond,
			600 * time.Millisecond,
		} {
			if delay > 0 {
				time.Sleep(delay)
			}
			overlay.applyNativeShape(radius)
		}
	}()
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
	height := screenSize.Height*overlayHeightFraction + overlayBottomClearance
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

func spriteScaleForResource(resource fyne.Resource) float32 {
	if isOversizedDirectionalSprite(resource) {
		return oversizedSpriteScale
	}
	return 1
}

func spriteTransformForResource(resource fyne.Resource) spriteTransform {
	scale := spriteScaleForResource(resource)
	transform := spriteTransform{
		scaleX: 1,
		scaleY: scale,
	}
	if isOversizedDirectionalSprite(resource) {
		transform.scaleX = oversizedSpriteWidth
		transform.scaleY = scale * oversizedSpriteHeight
		transform.offsetYFraction = oversizedSpriteOffsetY
		transform.stretch = true
	}
	return transform
}

func isOversizedDirectionalSprite(resource fyne.Resource) bool {
	if resource == nil {
		return false
	}
	name := resource.Name()
	return strings.HasSuffix(name, "Falcon looks down.png") || strings.HasSuffix(name, "Falcon looks left.png")
}

type spriteTransform struct {
	scaleX          float32
	scaleY          float32
	offsetYFraction float32
	stretch         bool
}

func normalizeSpriteTransform(transform spriteTransform) spriteTransform {
	if transform.scaleX <= 0 {
		transform.scaleX = 1
	}
	if transform.scaleY <= 0 {
		transform.scaleY = 1
	}
	return transform
}

type rightPanelLayout struct {
	spriteTransform spriteTransform
}

func (layout *rightPanelLayout) SetSpriteScale(scale float32) {
	layout.SetSpriteTransform(spriteTransform{scaleX: 1, scaleY: scale})
}

func (layout *rightPanelLayout) SetSpriteTransform(transform spriteTransform) {
	layout.spriteTransform = normalizeSpriteTransform(transform)
}

func (layout *rightPanelLayout) currentSpriteTransform() spriteTransform {
	return normalizeSpriteTransform(layout.spriteTransform)
}

func (layout *rightPanelLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 2 {
		return
	}
	image := objects[0]
	skip := objects[1]
	transform := layout.currentSpriteTransform()

	skipSize := skip.MinSize()
	if !skip.Visible() {
		baseSide := size.Height
		if baseSide > size.Width {
			baseSide = size.Width
		}
		height := baseSide * transform.scaleY
		width := height * transform.scaleX
		x := size.Width - width
		if x < 0 {
			x = 0
		}
		y := (size.Height-baseSide)/2 + (baseSide - height) + baseSide*transform.offsetYFraction
		if y < 0 {
			y = 0
		}
		image.Move(fyne.NewPos(x, y))
		image.Resize(fyne.NewSize(width, height))
		return
	}

	skipHeight := skipSize.Height
	if skipHeight > size.Height*0.25 {
		skipHeight = size.Height * 0.25
	}
	bottomClearance := overlayBottomClearance
	if bottomClearance > size.Height-skipHeight {
		bottomClearance = size.Height - skipHeight
	}
	if bottomClearance < 0 {
		bottomClearance = 0
	}
	imageAreaHeight := size.Height - skipHeight - bottomClearance
	if imageAreaHeight < 0 {
		imageAreaHeight = 0
	}

	margin := imageAreaHeight * 0.05
	baseSide := imageAreaHeight * 0.90 * skipModeImageScale
	if baseSide > size.Width-margin {
		baseSide = size.Width - margin
	}
	if baseSide < 0 {
		baseSide = 0
	}
	height := baseSide * transform.scaleY
	width := height * transform.scaleX
	if width > size.Width-margin {
		width = size.Width - margin
	}
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	x := size.Width - margin - width
	if x < 0 {
		x = 0
	}
	y := margin + (baseSide - height) + baseSide*transform.offsetYFraction
	image.Move(fyne.NewPos(x, y))
	image.Resize(fyne.NewSize(width, height))

	skipWidth := skipSize.Width
	skipWidth = skipWidth * 1.4
	if skipWidth > size.Width {
		skipWidth = size.Width
	}
	skipX := x + width - skipWidth
	if skipX < 0 {
		skipX = 0
	}
	skipY := imageAreaHeight + (skipHeight-skipSize.Height)/2
	maxSkipY := size.Height - bottomClearance - skipSize.Height
	if skipY > maxSkipY {
		skipY = maxSkipY
	}
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
	return fyne.NewSize(width, imageMin.Height+skipMin.Height+overlayBottomClearance)
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
