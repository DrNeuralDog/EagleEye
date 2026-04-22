package animation

import (
	"context"
	"crypto/rand"
	"math/big"
	"sync"
	"time"

	"fyne.io/fyne/v2"
)

// ExerciseType defines the type of exercise sequence.
type ExerciseType int

const (
	ExerciseLeftRight ExerciseType = iota
	ExerciseUpDown
	ExerciseBlink
	ExerciseLookOutside
)

// Range defines a duration range with random sampling.
type Range struct {
	Min time.Duration
	Max time.Duration
}

// Random returns a random duration within the range.
func (value Range) Random() time.Duration {
	if value.Max <= value.Min {
		return value.Min
	}
	delta := value.Max - value.Min
	offset, err := rand.Int(rand.Reader, big.NewInt(int64(delta)))
	if err != nil {
		return value.Min
	}
	return value.Min + time.Duration(offset.Int64())
}

// Config contains animation timing values.
type Config struct {
	InstructionDuration time.Duration

	CenterDuration Range
	MoveDuration   Range
	HoldDuration   Range
	ReturnDuration Range
	PauseDuration  Range

	BlinkClosedDuration Range
	BlinkOpenDuration   Range
	BlinkInterval       Range
	DoubleBlinkChance   float64
	DoubleBlinkGap      Range

	CombinedSwitchAfter time.Duration
}

// Engine manages sprite animations for the overlay.
type Engine struct {
	mu           sync.Mutex
	config       Config
	updateSprite func(fyne.Resource)
	onExercise   func(ExerciseType)
	cancel       context.CancelFunc
}

// New creates a new animation engine.
func New(config Config, updateSprite func(fyne.Resource)) *Engine {
	return &Engine{
		config:       config,
		updateSprite: updateSprite,
	}
}

// StartExercise starts an exercise animation sequence.
func (engine *Engine) StartExercise(ctx context.Context, spec ExerciseSpec) {
	engine.start(ctx, func(runCtx context.Context) {
		engine.updateSprite(spec.Instruction)
		if !sleepWithContext(runCtx, engine.config.InstructionDuration) {
			return
		}
		if spec.Duration > engine.config.InstructionDuration {
			spec.Duration -= engine.config.InstructionDuration
		} else {
			spec.Duration = 0
		}
		engine.runExercise(runCtx, spec)
	})
}

// StartIdle starts an idle animation loop.
func (engine *Engine) StartIdle(ctx context.Context, idle IdleSpec) {
	engine.start(ctx, func(runCtx context.Context) {
		engine.updateSprite(idle.Open)
		for {
			if !sleepWithContext(runCtx, engine.config.BlinkInterval.Random()) {
				return
			}
			engine.updateSprite(idle.Closed)
			if !sleepWithContext(runCtx, engine.config.BlinkClosedDuration.Random()) {
				return
			}
			engine.updateSprite(idle.Open)
			if !sleepWithContext(runCtx, engine.config.BlinkOpenDuration.Random()) {
				return
			}

			if randomUnitFloat() <= engine.config.DoubleBlinkChance {
				if !sleepWithContext(runCtx, engine.config.DoubleBlinkGap.Random()) {
					return
				}
				engine.updateSprite(idle.Closed)
				if !sleepWithContext(runCtx, engine.config.BlinkClosedDuration.Random()) {
					return
				}
				engine.updateSprite(idle.Open)
			}
		}
	})
}

// Stop terminates any active animation.
func (engine *Engine) Stop() {
	engine.mu.Lock()
	defer engine.mu.Unlock()
	if engine.cancel != nil {
		engine.cancel()
		engine.cancel = nil
	}
}

// SetOnExerciseChange sets a callback that is fired when active exercise changes.
func (engine *Engine) SetOnExerciseChange(handler func(ExerciseType)) {
	engine.mu.Lock()
	defer engine.mu.Unlock()
	engine.onExercise = handler
}

func (engine *Engine) start(parent context.Context, run func(context.Context)) {
	engine.mu.Lock()
	if engine.cancel != nil {
		engine.cancel()
	}
	runCtx, cancel := context.WithCancel(parent)
	engine.cancel = cancel
	engine.mu.Unlock()

	go run(runCtx)
}

func (engine *Engine) runExercise(ctx context.Context, spec ExerciseSpec) {
	if spec.Type == ExerciseLookOutside {
		engine.notifyExerciseChange(ExerciseLookOutside)
		engine.updateSprite(spec.LookOutside)
		<-ctx.Done()
		return
	}

	remaining := spec.Duration
	if remaining <= 0 {
		<-ctx.Done()
		return
	}

	if spec.Type == ExerciseLeftRight && remaining >= engine.config.CombinedSwitchAfter {
		segment := engine.config.CombinedSwitchAfter
		engine.notifyExerciseChange(ExerciseLeftRight)
		engine.runDirectional(ctx, spec, ExerciseLeftRight, segment)
		remaining -= segment
		if remaining > 0 {
			engine.notifyExerciseChange(ExerciseUpDown)
			engine.runDirectional(ctx, spec, ExerciseUpDown, remaining)
		}
		return
	}

	if spec.Type == ExerciseBlink {
		engine.notifyExerciseChange(ExerciseBlink)
		engine.runBlinkExercise(ctx, spec, remaining)
		return
	}

	engine.notifyExerciseChange(spec.Type)
	engine.runDirectional(ctx, spec, spec.Type, remaining)
}

func (engine *Engine) notifyExerciseChange(exercise ExerciseType) {
	engine.mu.Lock()
	handler := engine.onExercise
	engine.mu.Unlock()
	if handler != nil {
		handler(exercise)
	}
}

func (engine *Engine) runDirectional(ctx context.Context, spec ExerciseSpec, exercise ExerciseType, duration time.Duration) {
	start := time.Now()
	for time.Since(start) < duration {
		engine.updateSprite(spec.Center)
		if !sleepWithContext(ctx, engine.config.CenterDuration.Random()) {
			return
		}

		first := spec.Left
		second := spec.Right
		if exercise == ExerciseUpDown {
			first = spec.Up
			second = spec.Down
		}

		if !engine.runMove(ctx, first, spec.Center) {
			return
		}
		if !engine.runMove(ctx, second, spec.Center) {
			return
		}
	}
}

func (engine *Engine) runMove(ctx context.Context, target, center fyne.Resource) bool {
	engine.updateSprite(target)
	if !sleepWithContext(ctx, engine.config.MoveDuration.Random()) {
		return false
	}
	if !sleepWithContext(ctx, engine.config.HoldDuration.Random()) {
		return false
	}
	engine.updateSprite(center)
	if !sleepWithContext(ctx, engine.config.ReturnDuration.Random()) {
		return false
	}
	return sleepWithContext(ctx, engine.config.PauseDuration.Random())
}

func (engine *Engine) runBlinkExercise(ctx context.Context, spec ExerciseSpec, duration time.Duration) {
	longHold := true
	deadline := time.Now().Add(duration)
	engine.updateSprite(spec.BlinkOpen)
	for time.Now().Before(deadline) {
		if !sleepWithContext(ctx, spec.BlinkHoldDuration(longHold)) {
			return
		}
		engine.updateSprite(spec.BlinkClosed)
		if !sleepWithContext(ctx, spec.BlinkHoldDuration(longHold)) {
			return
		}
		engine.updateSprite(spec.BlinkOpen)
		longHold = !longHold
	}
}

func sleepWithContext(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func randomUnitFloat() float64 {
	const precision = int64(1_000_000)
	value, err := rand.Int(rand.Reader, big.NewInt(precision))
	if err != nil {
		return 0
	}
	return float64(value.Int64()) / float64(precision)
}
