//go:build windows

package cryptutil

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

const cryptProtectUIForbidden = 0x1

// Protect 使用当前 Windows 用户的 DPAPI 加密，无需额外密码。
func Protect(plaintext []byte) ([]byte, error) {
	in := newBlob(plaintext)
	var out windows.DataBlob
	if err := windows.CryptProtectData(&in, nil, nil, 0, nil, cryptProtectUIForbidden, &out); err != nil {
		return nil, fmt.Errorf("DPAPI 加密失败: %w", err)
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return blobBytes(out), nil
}

// Unprotect 解密本机当前用户 DPAPI 密文。换用户或换机器会失败。
func Unprotect(ciphertext []byte) ([]byte, error) {
	in := newBlob(ciphertext)
	var out windows.DataBlob
	if err := windows.CryptUnprotectData(&in, nil, nil, 0, nil, cryptProtectUIForbidden, &out); err != nil {
		return nil, fmt.Errorf("DPAPI 解密失败: %w", err)
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return blobBytes(out), nil
}

func newBlob(d []byte) windows.DataBlob {
	if len(d) == 0 {
		return windows.DataBlob{}
	}
	return windows.DataBlob{Data: &d[0], Size: uint32(len(d))}
}

func blobBytes(b windows.DataBlob) []byte {
	if b.Size == 0 || b.Data == nil {
		return []byte{}
	}
	return append([]byte(nil), unsafe.Slice(b.Data, int(b.Size))...)
}
