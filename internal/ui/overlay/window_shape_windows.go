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
// rectangular shape (used when switching into fullscreen mode)
//
// It uses the cached HWND populated by applyNativeOpacity /
// applyNativeTopmost rather than RunNative, so it can be safely retried
// from a goroutine after window.Show(): - that matches the working
// pattern used by forceForeground/scheduleInitialFocus
func (overlay *Window) applyNativeShape(cornerRadius int32) {
	hwnd := overlay.cachedHWND
	if hwnd == 0 {
		return
	}

	if cornerRadius <= 0 {
		// NULL region - removes any prior clipping
		_, _, _ = procSetWindowRgn.Call(hwnd, 0, 1)

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

	//Функция CreateRoundRectRgn принимает диаметр эллипса, используемого для отрисовки углов, а не радиус - поэтому удваивание значения
	ellipse := uintptr(cornerRadius * 2)
	rgn, _, _ := procCreateRoundRectRgn.Call(
		0, 0,
		uintptr(width)+1, uintptr(height)+1,
		ellipse, ellipse,
	)

	if rgn == 0 {
		return
	}

	// сли WinAPI-вызов прошёл успешно Windows сама дальше отвечает за этот handle.
	// Если вызов упал, handle остаётся твоей ответственностью и его надо закрыть самому!
	setRet, _, _ := procSetWindowRgn.Call(hwnd, rgn, 1)

	if setRet == 0 {
		_, _, _ = procDeleteObject.Call(rgn)
	}
}
