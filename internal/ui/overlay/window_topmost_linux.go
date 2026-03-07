//go:build linux

package overlay

import (
	"fmt"
	"os/exec"
	"sync"

	"eagleeye/internal/platform"

	"fyne.io/fyne/v2/driver"
)

var (
	wmctrlLookupOnce sync.Once
	wmctrlPath       string
)

func (overlay *Window) applyNativeTopmost(enable bool) {
	path, ok := lookupWmctrl()
	if !ok {
		return
	}

	nativeWindow, ok := overlay.window.(driver.NativeWindow)
	if !ok {
		return
	}

	nativeWindow.RunNative(func(context any) {
		x11WindowID := extractX11WindowID(context)
		if x11WindowID == 0 {
			return
		}

		action := "remove,above"
		if enable {
			action = "add,above"
		}

		windowID := fmt.Sprintf("0x%x", x11WindowID)
		_ = exec.Command(path, "-i", "-r", windowID, "-b", action).Run()
	})
}

func lookupWmctrl() (string, bool) {
	wmctrlLookupOnce.Do(func() {
		path, ok := platform.FindSystemExecutable("wmctrl")
		if !ok {
			return
		}
		wmctrlPath = path
	})
	if wmctrlPath == "" {
		return "", false
	}
	return wmctrlPath, true
}

func (overlay *Window) forceForeground() {}

func (overlay *Window) keepTopmost() {}

func (overlay *Window) releaseClipCursor() {}

func extractX11WindowID(context any) uintptr {
	switch value := context.(type) {
	case driver.X11WindowContext:
		return value.WindowHandle
	case *driver.X11WindowContext:
		return value.WindowHandle
	default:
		return 0
	}
}
