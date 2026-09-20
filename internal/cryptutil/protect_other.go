//go:build !windows

package cryptutil

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

var (
	fallbackOnce sync.Once
	fallbackKey  []byte
	fallbackErr  error
)

// Protect 在非 Windows 平台用本地随机密钥 + AES-GCM 模拟系统加密。
func Protect(plaintext []byte) ([]byte, error) {
	key, err := localKey()
	if err != nil {
		return nil, err
	}
	return EncryptAESGCM(key, plaintext)
}

// Unprotect 解密 Protect 产出的密文。
func Unprotect(ciphertext []byte) ([]byte, error) {
	key, err := localKey()
	if err != nil {
		return nil, err
	}
	return DecryptAESGCM(key, ciphertext)
}

func localKey() ([]byte, error) {
	fallbackOnce.Do(func() {
		dir, err := os.UserConfigDir()
		if err != nil {
			fallbackErr = err
			return
		}
		dir = filepath.Join(dir, "mfa_reader")
		if err := os.MkdirAll(dir, 0700); err != nil {
			fallbackErr = err
			return
		}
		keyPath := filepath.Join(dir, "local.key")
		if data, err := os.ReadFile(keyPath); err == nil && len(data) == aesKeySize {
			fallbackKey = data
			return
		}
		key := make([]byte, aesKeySize)
		if _, err := io.ReadFull(rand.Reader, key); err != nil {
			fallbackErr = err
			return
		}
		if err := os.WriteFile(keyPath, key, 0600); err != nil {
			fallbackErr = err
			return
		}
		fallbackKey = key
	})
	if fallbackErr != nil {
		return nil, fmt.Errorf("本地密钥: %w", fallbackErr)
	}
	return fallbackKey, nil
}
