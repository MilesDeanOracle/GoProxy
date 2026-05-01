//go:build windows

package platform

import (
	"syscall"
	"unsafe"
)

var (
	user32                = syscall.NewLazyDLL("user32.dll")
	procFindWindowW       = user32.NewProc("FindWindowW")
	procGetWindowLongPtrW = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtrW = user32.NewProc("SetWindowLongPtrW")
	procSetWindowPos      = user32.NewProc("SetWindowPos")
)

const (
	gwlStyle        = ^uintptr(15)
	wsMaximizebox   = 0x00010000
	swpFramechanged = 0x0020
	swpNomove       = 0x0002
	swpNosize       = 0x0001
	swpNozorder     = 0x0004
)

// DisableMaximizeButton removes the maximize button from the Wails window.
func DisableMaximizeButton(windowTitle string) {
	className, _ := syscall.UTF16PtrFromString("wailsWindow")
	title, _ := syscall.UTF16PtrFromString(windowTitle)

	hwnd, _, _ := procFindWindowW.Call(
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
	)
	if hwnd == 0 {
		hwnd, _, _ = procFindWindowW.Call(
			uintptr(unsafe.Pointer(className)),
			0,
		)
	}
	if hwnd == 0 {
		return
	}

	style, _, _ := procGetWindowLongPtrW.Call(hwnd, gwlStyle)
	newStyle := style & ^uintptr(wsMaximizebox)
	procSetWindowLongPtrW.Call(hwnd, gwlStyle, newStyle)

	procSetWindowPos.Call(
		hwnd, 0, 0, 0, 0, 0,
		uintptr(swpFramechanged|swpNomove|swpNosize|swpNozorder),
	)
}
