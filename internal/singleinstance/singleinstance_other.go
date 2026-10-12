//go:build !windows

package singleinstance

import "errors"

// ErrAlreadyRunning 表示同名实例已在运行。
var ErrAlreadyRunning = errors.New("已有实例在运行")

// Acquire 在非 Windows 平台不做单实例限制，始终允许启动。
func Acquire(string) (func(), error) {
	return func() {}, nil
}
