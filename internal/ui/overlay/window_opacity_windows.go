//go:build windows

package overlay

import (
	"syscall"

	"fyne.io/fyne/v2/driver"
)

const (
	gwlExStyle  int32 = -20
	wsExLayered       = 0x00080000
	lwaAlpha          = 0x2
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

		style, _, _ := procGetWindowLongPtrW.Call(hwnd, int32ToUintptr(gwlExStyle))
		if style&wsExLayered == 0 {
			procSetWindowLongPtrW.Call(hwnd, int32ToUintptr(gwlExStyle), style|wsExLayered)
		}
		procSetLayeredWindowAttributes.Call(hwnd, 0, uintptr(alpha), uintptr(lwaAlpha))
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

func int32ToUintptr(value int32) uintptr {
	return uintptr(uint32(value))
}
