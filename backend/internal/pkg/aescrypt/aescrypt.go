// Package aescrypt AES-256-GCM 加解密：用于敏感字段（如 API Secret）落库加密。
// 主密钥来自环境变量 SIM_ENC_KEY（任意长度口令，内部 SHA-256 派生 32 字节密钥）。
package aescrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

var ErrCiphertext = errors.New("密文格式错误或密钥不匹配")

type Cryptor struct {
	gcm cipher.AEAD
}

// New 由口令派生密钥并初始化。
func New(passphrase string) (*Cryptor, error) {
	key := sha256.Sum256([]byte(passphrase))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Cryptor{gcm: gcm}, nil
}

// Encrypt 输出 base64(nonce || ciphertext+tag)。
func (c *Cryptor) Encrypt(plaintext string) (string, error) {
	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("生成随机数失败: %w", err)
	}
	out := c.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(out), nil
}

// Decrypt 解密 base64(nonce || ciphertext+tag)。
func (c *Cryptor) Decrypt(encoded string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", ErrCiphertext
	}
	if len(raw) < c.gcm.NonceSize() {
		return "", ErrCiphertext
	}
	nonce, ciphertext := raw[:c.gcm.NonceSize()], raw[c.gcm.NonceSize():]
	plain, err := c.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", ErrCiphertext
	}
	return string(plain), nil
}
