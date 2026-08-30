package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"

	"cryptosim/internal/model"

	"gorm.io/gorm"
)

// issueCredentials 为用户签发（或重置）HMAC 验签凭证对。
// apiSecret 明文只在本返回值中出现一次，落库前用 AES-GCM 加密。
func issueCredentials(db *gorm.DB, cryptor SecretCryptor, uid int64) (apiKey, apiSecret string, err error) {
	apiKey, apiSecret = randomToken("csk_", 16), randomToken("css_", 32)
	enc, err := cryptor.Encrypt(apiSecret)
	if err != nil {
		return "", "", err
	}
	var cred model.ApiCredential
	err = db.Where("user_id = ?", uid).First(&cred).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		cred = model.ApiCredential{UserID: uid, ApiKey: apiKey, SecretEncrypted: enc}
		err = db.Create(&cred).Error
	case err != nil:
		return "", "", err
	default:
		err = db.Model(&cred).Updates(map[string]any{
			"api_key": apiKey, "secret_encrypted": enc,
		}).Error
	}
	if err != nil {
		return "", "", err
	}
	return apiKey, apiSecret, nil
}

func randomToken(prefix string, nBytes int) string {
	b := make([]byte, nBytes)
	_, _ = rand.Read(b)
	return prefix + hex.EncodeToString(b)
}

// SecretCryptor 敏感字段加解密能力（由 aescrypt 实现）。
type SecretCryptor interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(encoded string) (string, error)
}
