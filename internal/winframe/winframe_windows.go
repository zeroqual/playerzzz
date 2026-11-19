//go:build windows

package winframe

import (
	"sync/atomic"
	"syscall"
	"unsafe"
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procSetWindowLongPtrW   = user32.NewProc("SetWindowLongPtrW")
	procCallWindowProcW     = user32.NewProc("CallWindowProcW")
	procSetWindowPos        = user32.NewProc("SetWindowPos")
	procGetWindowRect       = user32.NewProc("GetWindowRect")
	procIsZoomed            = user32.NewProc("IsZoomed")
	procShowWindow          = user32.NewProc("ShowWindow")
	procGetDpiForWindow     = user32.NewProc("GetDpiForWindow")
	procGetSystemMetricsDpi = user32.NewProc("GetSystemMetricsForDpi")
	procGetSystemMetrics    = user32.NewProc("GetSystemMetrics")
)

const (
	gwlpWndProc = ^uintptr(3)

	wmNCCalcSize = 0x0083
	wmNCHitTest  = 0x0084

	htClient      = 1
	htCaption     = 2
	htLeft        = 10
	htRight       = 11
	htTop         = 12
	htTopLeft     = 13
	htTopRight    = 14
	htBottom      = 15
	htBottomLeft  = 16
	htBottomRight = 17

	swpNoSize       = 0x1
	swpNoMove       = 0x2
	swpNoZOrder     = 0x4
	swpNoActivate   = 0x10
	swpFrameChanged = 0x20

	swMaximize = 3
	swMinimize = 6
	swRestore  = 9

	smCxFrame        = 32
	smCxPaddedBorder = 92
)

type rect struct{ Left, Top, Right, Bottom int32 }

type ncCalcSizeParams struct {
	rgrc  [3]rect
	lppos uintptr
}

var (
	oldProc   uintptr
	captionPx atomic.Int32
	buttonsPx atomic.Int32
	callback  = syscall.NewCallback(wndProc)
)

func Supported() bool { return true }

func Enable(hwnd uintptr) error {
	if hwnd == 0 || oldProc != 0 {
		return nil
	}
	prev, _, err := procSetWindowLongPtrW.Call(hwnd, gwlpWndProc, callback)
	if prev == 0 {
		return err
	}
	oldProc = prev

	procSetWindowPos.Call(hwnd, 0, 0, 0, 0, 0,
		swpNoMove|swpNoSize|swpNoZOrder|swpNoActivate|swpFrameChanged)
	return nil
}

func SetMetrics(caption, buttons int32) {
	captionPx.Store(caption)
	buttonsPx.Store(buttons)
}

func Minimize(hwnd uintptr) { procShowWindow.Call(hwnd, swMinimize) }

func ToggleMaximize(hwnd uintptr) {
	if IsMaximized(hwnd) {
		procShowWindow.Call(hwnd, swRestore)
	} else {
		procShowWindow.Call(hwnd, swMaximize)
	}
}

func IsMaximized(hwnd uintptr) bool {
	r, _, _ := procIsZoomed.Call(hwnd)
	return r != 0
}

func wndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	switch msg {
	case wmNCCalcSize:
		if wParam != 0 {
			if IsMaximized(hwnd) {
				p := *(**ncCalcSizeParams)(unsafe.Pointer(&lParam))
				b := frameThickness(hwnd)
				p.rgrc[0].Left += b
				p.rgrc[0].Top += b
				p.rgrc[0].Right -= b
				p.rgrc[0].Bottom -= b
			}
			return 0
		}
	case wmNCHitTest:
		return hitTest(hwnd, lParam)
	}
	r, _, _ := procCallWindowProcW.Call(oldProc, hwnd, msg, wParam, lParam)
	return r
}

func hitTest(hwnd, lParam uintptr) uintptr {
	x := int32(int16(lParam & 0xFFFF))
	y := int32(int16((lParam >> 16) & 0xFFFF))
	var r rect
	procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))

	if !IsMaximized(hwnd) {
		b := frameThickness(hwnd)
		left, right := x < r.Left+b, x >= r.Right-b
		top, bottom := y < r.Top+b, y >= r.Bottom-b
		switch {
		case top && left:
			return htTopLeft
		case top && right:
			return htTopRight
		case bottom && left:
			return htBottomLeft
		case bottom && right:
			return htBottomRight
		case left:
			return htLeft
		case right:
			return htRight
		case top:
			return htTop
		case bottom:
			return htBottom
		}
	}
	if y < r.Top+captionPx.Load() && x < r.Right-buttonsPx.Load() {
		return htCaption
	}
	return htClient
}

func frameThickness(hwnd uintptr) int32 {
	if procGetDpiForWindow.Find() == nil && procGetSystemMetricsDpi.Find() == nil {
		dpi, _, _ := procGetDpiForWindow.Call(hwnd)
		f, _, _ := procGetSystemMetricsDpi.Call(smCxFrame, dpi)
		p, _, _ := procGetSystemMetricsDpi.Call(smCxPaddedBorder, dpi)
		return int32(f + p)
	}
	f, _, _ := procGetSystemMetrics.Call(smCxFrame)
	p, _, _ := procGetSystemMetrics.Call(smCxPaddedBorder)
	return int32(f + p)
}
