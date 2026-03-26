//go:build windows

package overlay

import "fyne.io/fyne/v2/driver"

const (
	hwndTopmost   = ^uintptr(0)
	hwndNoTopmost = ^uintptr(1)
	swpNoSize     = 0x0001
	swpNoMove     = 0x0002
	swpNoActivate = 0x0010
)

var procSetWindowPos = user32DLL.NewProc("SetWindowPos")

func (overlay *Window) applyNativeTopmost(enable bool) {
	nativeWindow, ok := overlay.window.(driver.NativeWindow)
	if !ok {
		return
	}

	nativeWindow.RunNative(func(context any) {
		var hwnd uintptr
		switch value := context.(type) {
		case driver.WindowsWindowContext:
			hwnd = value.HWND
		case *driver.WindowsWindowContext:
			hwnd = value.HWND
		default:
			return
		}
		if hwnd == 0 {
			return
		}

		insertAfter := hwndNoTopmost
		if enable {
			insertAfter = hwndTopmost
		}
		flags := uintptr(swpNoMove | swpNoSize | swpNoActivate)
		procSetWindowPos.Call(hwnd, insertAfter, 0, 0, 0, 0, flags)
	})
}
