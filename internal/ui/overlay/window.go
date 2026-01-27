package overlay

import (
	"context"
	"fmt"
	"image/color"
	"time"

	"eagleeye/internal/ui/animation"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
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
	app        fyne.App
	window     fyne.Window
	config     Config
	image      *canvas.Image
	timerLabel *widget.Label
	skipButton *widget.Button
	message    *widget.Label
	background *canvas.Rectangle
	engine     *animation.Engine
	cancelCtx  context.CancelFunc
	onSkip     func()
}

// New creates a new overlay window.
func New(app fyne.App, config Config, engine *animation.Engine) *Window {
	window := app.NewWindow("EagleEye")
	if app.Icon() != nil {
		window.SetIcon(app.Icon())
	}
	window.SetPadded(false)

	background := canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 0, A: config.Opacity})

	image := canvas.NewImageFromResource(nil)
	image.FillMode = canvas.ImageFillContain

	timerLabel := widget.NewLabel("--:--")
	timerLabel.Alignment = fyne.TextAlignLeading

	message := widget.NewLabel(config.Message)
	message.Alignment = fyne.TextAlignCenter
	message.TextStyle = fyne.TextStyle{Bold: true}

	skipButton := widget.NewButton("Skip", nil)

	bottom := container.NewHBox(timerLabel, layout.NewSpacer(), skipButton)
	content := container.NewBorder(message, bottom, nil, nil, container.NewCenter(image))
	root := container.NewMax(background, content)

	window.SetContent(root)
	if config.Fullscreen {
		window.SetFullScreen(true)
	}

	return &Window{
		app:        app,
		window:     window,
		config:     config,
		image:      image,
		timerLabel: timerLabel,
		skipButton: skipButton,
		message:    message,
		background: background,
		engine:     engine,
	}
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

	fyne.Do(func() {
		overlay.setRemainingUnsafe(session.Remaining)
		overlay.setStrictModeUnsafe(session.StrictMode)
		overlay.window.Show()
		overlay.window.RequestFocus()
	})

	if overlay.engine != nil {
		overlay.engine.StartExercise(ctx, spec)
	}
}

// ShowIdle starts the idle animation (long breaks).
func (overlay *Window) ShowIdle(remaining time.Duration, strict bool, idle animation.IdleSpec) {
	overlay.stopEngine()
	ctx, cancel := context.WithCancel(context.Background())
	overlay.cancelCtx = cancel

	fyne.Do(func() {
		overlay.setRemainingUnsafe(remaining)
		overlay.setStrictModeUnsafe(strict)
		overlay.window.Show()
		overlay.window.RequestFocus()
	})

	if overlay.engine != nil {
		overlay.engine.StartIdle(ctx, idle)
	}
}

// Hide closes the overlay and stops animations.
func (overlay *Window) Hide() {
	overlay.stopEngine()
	fyne.Do(func() {
		overlay.window.Hide()
	})
}

// SetRemaining updates the timer label.
func (overlay *Window) SetRemaining(remaining time.Duration) {
	overlay.setRemaining(remaining)
}

// SetStrictMode toggles skip visibility.
func (overlay *Window) SetStrictMode(enabled bool) {
	overlay.setStrictMode(enabled)
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
	fyne.Do(func() {
		overlay.config = config
		overlay.background.FillColor = color.NRGBA{R: 0, G: 0, B: 0, A: config.Opacity}
		overlay.message.SetText(config.Message)
		if config.Fullscreen {
			overlay.window.SetFullScreen(true)
		}
		canvas.Refresh(overlay.background)
		canvas.Refresh(overlay.message)
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
	fyne.Do(func() {
		overlay.setRemainingUnsafe(remaining)
	})
}

func (overlay *Window) setStrictMode(enabled bool) {
	fyne.Do(func() {
		overlay.setStrictModeUnsafe(enabled)
	})
}

func (overlay *Window) setRemainingUnsafe(remaining time.Duration) {
	overlay.timerLabel.SetText(formatDuration(remaining))
}

func (overlay *Window) setStrictModeUnsafe(enabled bool) {
	if enabled {
		overlay.skipButton.Disable()
		return
	}
	overlay.skipButton.Enable()
}

func (overlay *Window) stopEngine() {
	if overlay.cancelCtx != nil {
		overlay.cancelCtx()
		overlay.cancelCtx = nil
	}
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
