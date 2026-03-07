//go:build windows

package overlay

import (
	"syscall"
	"unsafe"
)

var (
	gdi32DLL               = syscall.NewLazyDLL("gdi32.dll")
	procCreateRoundRectRgn = gdi32DLL.NewProc("CreateRoundRectRgn")
	procDeleteObject       = gdi32DLL.NewProc("DeleteObject")
	procSetWindowRgn       = user32DLL.NewProc("SetWindowRgn")
	procGetClientRect      = user32DLL.NewProc("GetClientRect")
)

// applyNativeShape clips the native HWND to a rounded rectangle so the
// overlay window actually has rounded corners at the OS level. A zero
// (or negative) radius removes any prior region, restoring the default
// rectangular shape (used when switching into fullscreen mode).
//
// It uses the cached HWND populated by applyNativeOpacity /
// applyNativeTopmost rather than RunNative, so it can be safely retried
// from a goroutine after window.Show(): that matches the working
// pattern used by forceForeground/scheduleInitialFocus.
func (overlay *Window) applyNativeShape(cornerRadius int32) {
	hwnd := overlay.cachedHWND
	if hwnd == 0 {
		return
	}

	if cornerRadius <= 0 {
		// NULL region removes any prior clipping.
		procSetWindowRgn.Call(hwnd, 0, 1)
		return
	}

	var rect winRECT
	ret, _, _ := procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))
	if ret == 0 {
		return
	}
	width := rect.Right - rect.Left
	height := rect.Bottom - rect.Top
	if width <= 0 || height <= 0 {
		return
	}

	// CreateRoundRectRgn takes the diameter of the ellipse used to draw
	// the corners, not the radius, so double the value.
	ellipse := uintptr(cornerRadius * 2)
	rgn, _, _ := procCreateRoundRectRgn.Call(
		0, 0,
		uintptr(width)+1, uintptr(height)+1,
		ellipse, ellipse,
	)
	if rgn == 0 {
		return
	}
	// On success the system takes ownership of the region handle; on
	// failure we still own it and must free it ourselves to avoid a
	// GDI leak across retries.
	setRet, _, _ := procSetWindowRgn.Call(hwnd, rgn, 1)
	if setRet == 0 {
		procDeleteObject.Call(rgn)
	}
}
