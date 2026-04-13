//go:build !windows && !linux

package overlay

func (overlay *Window) applyNativeTopmost(enable bool) {
	_ = enable
}

func (overlay *Window) forceForeground() {}

func (overlay *Window) releaseClipCursor() {}

