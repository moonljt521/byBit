package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"strconv"
	"strings"
	"time"

	"cryptosim/internal/model"
	"cryptosim/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SecretCryptor 敏感字段加解密能力（与 auth 包约定一致，由 aescrypt 实现）。
type SecretCryptor interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(encoded string) (string, error)
}

// 签名规范（与前端/Flutter 客户端约定一致）：
//   headers: X-API-KEY / X-API-TIMESTAMP(unix 秒) / X-API-SIGNATURE
//   stringToSign = timestamp \n METHOD \n path \n rawQuery \n sha256hex(body)
//   X-API-SIGNATURE = hex(HMAC-SHA256(apiSecret, stringToSign))
//   时钟偏差容忍 ±300 秒（防重放）

const signWindow = 300 * time.Second

// Private 私有接口统一鉴权：JWT 身份 + HMAC 请求验签（防篡改/防重放）。
func Private(jwtSecret string, db *gorm.DB, cryptor SecretCryptor) gin.HandlerFunc {
	return privateAuth(jwtSecret, db, cryptor, false)
}

// PrivateAdmin 管理端鉴权：在 Private 基础上要求 admin 角色。
func PrivateAdmin(jwtSecret string, db *gorm.DB, cryptor SecretCryptor) gin.HandlerFunc {
	return privateAuth(jwtSecret, db, cryptor, true)
}

func privateAuth(jwtSecret string, db *gorm.DB, cryptor SecretCryptor, adminOnly bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1) JWT 身份
		uid, ok := parseUID(c, jwtSecret)
		if !ok {
			c.AbortWithStatusJSON(401, gin.H{"code": response.CodeUnauthorized, "msg": "未登录或凭证无效", "data": nil})
			return
		}
		// 2) API 凭证
		apiKey := c.GetHeader("X-API-KEY")
		ts := c.GetHeader("X-API-TIMESTAMP")
		sig := c.GetHeader("X-API-SIGNATURE")
		if apiKey == "" || ts == "" || sig == "" {
			abortCode(c, 401, "缺少签名头（X-API-KEY/TIMESTAMP/SIGNATURE）")
			return
		}
		tsInt, err := strconv.ParseInt(ts, 10, 64)
		if err != nil || time.Since(time.Unix(tsInt, 0)).Abs() > signWindow {
			abortCode(c, 401, "签名时间戳过期或非法（防重放）")
			return
		}
		var cred model.ApiCredential
		if err := db.Where("user_id = ?", uid).First(&cred).Error; err != nil {
			abortCode(c, 401, "API 凭证不存在，请重新登录")
			return
		}
		if apiKey != cred.ApiKey {
			abortCode(c, 401, "API Key 不匹配")
			return
		}
		secret, err := cryptor.Decrypt(cred.SecretEncrypted)
		if err != nil {
			abortCode(c, 401, "凭证解密失败，请重置 API 凭证")
			return
		}
		// 3) 读 body（读出后回填，业务 handler 仍可正常解析）
		var body []byte
		if c.Request.Body != nil {
			body, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewReader(body))
		}
		// 4) 验签
		bodyHash := sha256.Sum256(body)
		sts := strings.Join([]string{
			ts,
			c.Request.Method,
			c.Request.URL.Path,
			c.Request.URL.RawQuery,
			hex.EncodeToString(bodyHash[:]),
		}, "\n")
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(sts))
		expect := hex.EncodeToString(mac.Sum(nil))
		if !hmac.Equal([]byte(expect), []byte(strings.ToLower(sig))) {
			abortCode(c, 401, "签名校验失败")
			return
		}
		// 5) 角色校验（管理端）
		if adminOnly {
			var u model.User
			if err := db.Select("id", "role", "status").First(&u, uid).Error; err != nil || u.Role != "admin" {
				abortCode(c, 403, "需要管理员权限")
				return
			}
			if u.Status != 1 {
				abortCode(c, 403, "账号已被禁用")
				return
			}
		}
		c.Set(ctxUID, uid)
		c.Next()
	}
}

func abortCode(c *gin.Context, status int, msg string) {
	c.AbortWithStatusJSON(status, gin.H{"code": response.CodeUnauthorized, "msg": msg, "data": nil})
}
