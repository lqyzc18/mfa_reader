package cryptutil

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/scrypt"
)

const (
	aesKeySize  = 32
	scryptN     = 1 << 15
	scryptR     = 8
	scryptP     = 1
	saltSize    = 16
	minPassword = 6
)

var (
	ErrShortCipher   = errors.New("密文过短")
	ErrShortPassword = fmt.Errorf("主密码至少 %d 位", minPassword)
)

// EncryptAESGCM 使用 32 字节密钥加密，返回 nonce||ciphertext。
func EncryptAESGCM(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// DecryptAESGCM 解密 nonce||ciphertext。
func DecryptAESGCM(key, blob []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	ns := gcm.NonceSize()
	if len(blob) < ns {
		return nil, ErrShortCipher
	}
	return gcm.Open(nil, blob[:ns], blob[ns:], nil)
}

// NewSalt 生成 scrypt 盐。
func NewSalt() ([]byte, error) {
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	return salt, nil
}

// DeriveKey 用 scrypt 从主密码派生 AES-256 密钥。
func DeriveKey(password string, salt []byte) ([]byte, error) {
	if len(password) < minPassword {
		return nil, ErrShortPassword
	}
	return scrypt.Key([]byte(password), salt, scryptN, scryptR, scryptP, aesKeySize)
}
