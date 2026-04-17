//go:build windows

package overlay

import (
	"unsafe"

	"fyne.io/fyne/v2/driver"
)

const (
	hwndTopmost   = ^uintptr(0)
	hwndNoTopmost = ^uintptr(1)
	swpNoSize     = 0x0001
	swpNoMove     = 0x0002
	swpNoActivate = 0x0010
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
		procSetWindowPos.Call(hwnd, insertAfter, 0, 0, 0, 0, flags)
	})
}

// forceForeground re-claims focus, re-applies topmost + opacity,
// and clips the cursor to the overlay window so the user cannot
// interact with other screens during a strict-mode break.
func (overlay *Window) forceForeground() {
	hwnd := overlay.cachedHWND
	if hwnd == 0 {
		return
	}

	procSetForegroundWindow.Call(hwnd)
	procSetWindowPos.Call(hwnd, hwndTopmost, 0, 0, 0, 0, uintptr(swpNoMove|swpNoSize))

	// Re-apply layered opacity in case Windows reset it.
	style, _, _ := procGetWindowLongPtrW.Call(hwnd, int32ToUintptr(gwlExStyle))
	if style&wsExLayered == 0 {
		procSetWindowLongPtrW.Call(hwnd, int32ToUintptr(gwlExStyle), style|wsExLayered)
	}
	procSetLayeredWindowAttributes.Call(hwnd, 0, uintptr(overlay.config.Opacity), uintptr(lwaAlpha))

	// Clip cursor only in strict mode.
	if overlay.strictMode {
		var rect winRECT
		procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))
		procClipCursor.Call(uintptr(unsafe.Pointer(&rect)))
	}
}

// keepTopmost restores topmost ordering without activating the overlay.
func (overlay *Window) keepTopmost() {
	hwnd := overlay.cachedHWND
	if hwnd == 0 {
		return
	}
	procSetWindowPos.Call(hwnd, hwndTopmost, 0, 0, 0, 0, uintptr(swpNoMove|swpNoSize|swpNoActivate))
}

// releaseClipCursor removes the cursor restriction.
func (overlay *Window) releaseClipCursor() {
	procClipCursor.Call(0)
}
