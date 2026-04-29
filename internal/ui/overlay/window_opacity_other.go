//go:build !windows

package overlay

// applyNativeOpacity is a no-op on platforms without native opacity support
func (overlay *Window) applyNativeOpacity(alpha uint8) {
	_ = alpha
}
