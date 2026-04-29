//go:build windows

package overlay

import (
	"unsafe"

	"fyne.io/fyne/v2/driver"
)

const (
	hwndTopmost   uintptr = ^uintptr(0)
	hwndNoTopmost uintptr = ^uintptr(1)
	swpNoSize     uintptr = 0x0001
	swpNoMove     uintptr = 0x0002
	swpNoActivate uintptr = 0x0010
)

type winRECT struct {
	Left, Top, Right, Bottom int32
}

var (
	procSetWindowPos        = user32DLL.NewProc("SetWindowPos")
	procSetForegroundWindow = user32DLL.NewProc("SetForegroundWindow")
	procGetWindowRect       = user32DLL.NewProc("GetWindowRect")
	procClipCursor          = user32DLL.NewProc("ClipCursor")
)

func (overlay *Window) applyNativeTopmost(enable bool) {
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

		insertAfter := hwndNoTopmost

		if enable {
			insertAfter = hwndTopmost
		}

		flags := uintptr(swpNoMove | swpNoSize | swpNoActivate)
		_, _, _ = procSetWindowPos.Call(hwnd, insertAfter, 0, 0, 0, 0, flags)
	})
}

// forceForeground re-claims focus, re-applies topmost + opacity,
// and clips the cursor to the overlay window so the user cannot
// interact with other screens during a strict-mode break
func (overlay *Window) forceForeground() {
	hwnd := overlay.cachedHWND

	if hwnd == 0 {
		return
	}

	_, _, _ = procSetForegroundWindow.Call(hwnd)
	_, _, _ = procSetWindowPos.Call(hwnd, hwndTopmost, 0, 0, 0, 0, uintptr(swpNoMove|swpNoSize))

	// Re-apply layered opacity in case Windows reset it.
	style, _, _ := procGetWindowLongPtrW.Call(hwnd, gwlExStyle)

	if style&wsExLayered == 0 {
		_, _, _ = procSetWindowLongPtrW.Call(hwnd, gwlExStyle, style|wsExLayered)
	}
	_, _, _ = procSetLayeredWindowAttributes.Call(hwnd, 0, uintptr(overlay.config.Opacity), lwaAlpha)

	// Clip cursor only in strict mode
	if overlay.strictMode {
		var rect winRECT
		ret, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))

		if ret != 0 {
			_, _, _ = procClipCursor.Call(uintptr(unsafe.Pointer(&rect)))
		}
	}
}

// keepTopmost restores topmost ordering without activating the overlay
func (overlay *Window) keepTopmost() {
	hwnd := overlay.cachedHWND

	if hwnd == 0 {
		return
	}
	_, _, _ = procSetWindowPos.Call(hwnd, hwndTopmost, 0, 0, 0, 0, uintptr(swpNoMove|swpNoSize|swpNoActivate))
}

// releaseClipCursor removes the cursor restriction
func (overlay *Window) releaseClipCursor() {
	_, _, _ = procClipCursor.Call(0)
}
