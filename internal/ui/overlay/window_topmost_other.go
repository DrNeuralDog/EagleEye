//go:build !windows && !linux

package overlay

func (overlay *Window) applyNativeTopmost(enable bool) {
	_ = enable
}
