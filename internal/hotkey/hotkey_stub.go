//go:build !windows

package hotkey

// Start 非 Windows 平台不注册全局热键。
func Start(func()) (func(), error) {
	return func() {}, nil
}
