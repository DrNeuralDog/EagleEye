//go:build !windows

package overlay

func (overlay *Window) applyNativeOpacity(alpha uint8) {
	_ = alpha
}
