package overlay

import (
	"context"
	"time"
)

// scheduleInitialFocus re-applies native attributes shortly after the
// window appears, giving the OS time to finish compositing - this avoids
// the brief transparent flash on first show!
func (overlay *Window) scheduleInitialFocus(ctx context.Context) {
	go func() {
		for _, delay := range []time.Duration{
			50 * time.Millisecond,
			150 * time.Millisecond,
			300 * time.Millisecond,
		} {
			if !sleepOverlaySchedule(ctx, delay) {
				return
			}

			overlay.forceForeground()
		}
	}()
}

// scheduleTopmostKeep periodically restores the overlay topmost state for the
// lifetime of the active break session.
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
// few times with small delays after window.Show() - Retries are needed
// because the HWND can take a moment to receive its final client size
// via Windows message pumping, and because Fyne's initial layout may
// trigger an additional resize that clears the region.
func (overlay *Window) scheduleNativeShape(ctx context.Context, radius int32) {
	go func() {
		for _, delay := range []time.Duration{
			0,
			50 * time.Millisecond,
			150 * time.Millisecond,
			300 * time.Millisecond,
			600 * time.Millisecond,
		} {
			if !sleepOverlaySchedule(ctx, delay) {
				return
			}

			overlay.applyNativeShape(radius)
		}
	}()
}

// sleepOverlaySchedule waits for a retry delay while still allowing the
// scheduled overlay update to stop immediately when its context is cancelled
func sleepOverlaySchedule(ctx context.Context, delay time.Duration) bool {
	if delay <= 0 {
		select {
		case <-ctx.Done():
			return false
		default:
			return true
		}
	}

	timer := time.NewTimer(delay)

	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
