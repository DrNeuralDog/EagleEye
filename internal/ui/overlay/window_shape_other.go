//go:build !windows

package overlay

// applyNativeShape is a no-op on non-Windows platforms. Rounded window
// shapes there are handled (or not) by the window manager, and Fyne's
// canvas-level CornerRadius on the card background provides the visual
// rounding within the window frame
func (overlay *Window) applyNativeShape(cornerRadius int32) {
	_ = cornerRadius
}
