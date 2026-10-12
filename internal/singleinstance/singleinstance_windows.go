//go:build windows

package singleinstance

import (
	"errors"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ErrAlreadyRunning 表示同名实例已在运行。
var ErrAlreadyRunning = errors.New("已有实例在运行")

var (
	kernel32        = windows.NewLazySystemDLL("kernel32.dll")
	procCreateMutex = kernel32.NewProc("CreateMutexW")
)

// Acquire 用命名互斥量保证同一时间只有一个实例在运行，返回释放函数。
// 名称带 "Local\" 前缀：同一登录会话内互斥，不同用户会话互不干扰。
func Acquire(name string) (func(), error) {
	ptr, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}
	handle, _, callErr := procCreateMutex.Call(0, 0, uintptr(unsafe.Pointer(ptr)))
	if handle == 0 {
		return nil, callErr
	}
	if callErr == windows.ERROR_ALREADY_EXISTS {
		windows.CloseHandle(windows.Handle(handle))
		return nil, ErrAlreadyRunning
	}

	var once sync.Once
	return func() {
		once.Do(func() {
			windows.CloseHandle(windows.Handle(handle))
		})
	}, nil
}
