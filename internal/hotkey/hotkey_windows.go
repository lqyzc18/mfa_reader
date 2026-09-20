//go:build windows

package hotkey

import (
	"runtime"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	modControl = 0x0002
	modAlt     = 0x0001
	vkM        = 0x4D
	wmHotkey   = 0x0312
	hotkeyID   = 0x4D46 // "MF"
)

var (
	user32           = windows.NewLazySystemDLL("user32.dll")
	procRegister     = user32.NewProc("RegisterHotKey")
	procUnregister   = user32.NewProc("UnregisterHotKey")
	procGetMessage   = user32.NewProc("GetMessageW")
	procPostThread   = user32.NewProc("PostThreadMessageW")
	procGetCurrentID = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetCurrentThreadId")
)

type msg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      struct{ x, y int32 }
}

// Start 注册 Ctrl+Alt+M，在独立线程接收热键。返回取消函数。
func Start(onHotkey func()) (func(), error) {
	done := make(chan struct{})
	errCh := make(chan error, 1)
	var threadID uint32

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		r, _, err := procRegister.Call(0, hotkeyID, modControl|modAlt, vkM)
		if r == 0 {
			errCh <- err
			return
		}
		tid, _, _ := procGetCurrentID.Call()
		threadID = uint32(tid)
		errCh <- nil

		var m msg
		for {
			select {
			case <-done:
				procUnregister.Call(0, hotkeyID)
				return
			default:
			}
			ret, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
			if int32(ret) <= 0 {
				procUnregister.Call(0, hotkeyID)
				return
			}
			if m.message == wmHotkey && m.wParam == hotkeyID && onHotkey != nil {
				onHotkey()
			}
		}
	}()

	if err := <-errCh; err != nil {
		return func() {}, err
	}

	var once sync.Once
	return func() {
		once.Do(func() {
			close(done)
			if threadID != 0 {
				const wmQuit = 0x0012
				procPostThread.Call(uintptr(threadID), wmQuit, 0, 0)
			}
			procUnregister.Call(0, hotkeyID)
		})
	}, nil
}
