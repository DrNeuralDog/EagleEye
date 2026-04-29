//go:build !windows && !linux

package overlay

// applyNativeTopmost is a no-op on platforms without native topmost support!
func (overlay *Window) applyNativeTopmost(enable bool) {
	_ = enable
}

func (overlay *Window) forceForeground() {}

func (overlay *Window) keepTopmost() {}

func (overlay *Window) releaseClipCursor() {}
