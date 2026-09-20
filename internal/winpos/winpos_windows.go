//go:build windows

package winpos

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32         = windows.NewLazySystemDLL("user32.dll")
	procFindWindow = user32.NewProc("FindWindowW")
	procGetRect    = user32.NewProc("GetWindowRect")
	procSetPos     = user32.NewProc("SetWindowPos")
)

type rect struct {
	left, top, right, bottom int32
}

const (
	swpNoZOrder = 0x0004
	swpNoSize   = 0x0001
)

// Get 按窗口标题读取当前位置。
func Get(title string) (x, y int, ok bool) {
	hwnd := find(title)
	if hwnd == 0 {
		return 0, 0, false
	}
	var r rect
	ret, _, _ := procGetRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	if ret == 0 {
		return 0, 0, false
	}
	return int(r.left), int(r.top), true
}

// Set 按窗口标题移动窗口，保留当前尺寸。
func Set(title string, x, y int) {
	hwnd := find(title)
	if hwnd == 0 {
		return
	}
	procSetPos.Call(hwnd, 0, uintptr(x), uintptr(y), 0, 0, swpNoZOrder|swpNoSize)
}

func find(title string) uintptr {
	ptr, err := windows.UTF16PtrFromString(title)
	if err != nil {
		return 0
	}
	hwnd, _, _ := procFindWindow.Call(0, uintptr(unsafe.Pointer(ptr)))
	return hwnd
}
