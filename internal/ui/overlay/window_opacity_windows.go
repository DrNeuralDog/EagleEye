//go:build windows

package overlay

import (
	"syscall"

	"fyne.io/fyne/v2/driver"
)

const (
	gwlExStyle  uintptr = ^uintptr(19)
	wsExLayered uintptr = 0x00080000
	lwaAlpha    uintptr = 0x2
)

var (
	user32DLL                      = syscall.NewLazyDLL("user32.dll")
	procGetWindowLongPtrW          = user32DLL.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtrW          = user32DLL.NewProc("SetWindowLongPtrW")
	procSetLayeredWindowAttributes = user32DLL.NewProc("SetLayeredWindowAttributes")
)

func (overlay *Window) applyNativeOpacity(alpha uint8) {
	nativeWindow, ok := overlay.window.(driver.NativeWindow)

	if !ok {
		return
	}

	nativeWindow.RunNative(func(context any) {
		hwnd := extractHWND(context)

		if hwnd == 0 {
			return
		}

		overlay.cachedHWND = hwnd

		style, _, _ := procGetWindowLongPtrW.Call(hwnd, gwlExStyle)

		if style&wsExLayered == 0 {
			_, _, _ = procSetWindowLongPtrW.Call(hwnd, gwlExStyle, style|wsExLayered)
		}

		_, _, _ = procSetLayeredWindowAttributes.Call(hwnd, 0, uintptr(alpha), lwaAlpha)
	})
}

func extractHWND(context any) uintptr {
	switch value := context.(type) {
	case driver.WindowsWindowContext:
		return value.HWND
	case *driver.WindowsWindowContext:
		return value.HWND
	default:
		return 0
	}
}
