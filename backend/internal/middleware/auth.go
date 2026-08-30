// Package middleware Gin 中间件：JWT 鉴权与管理员鉴权。
package middleware

import (
	"strings"

	"cryptosim/internal/model"
	"cryptosim/internal/pkg/jwtutil"
	"cryptosim/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const ctxUID = "uid"

func parseUID(c *gin.Context, secret string) (int64, bool) {
	header := c.GetHeader("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return 0, false
	}
	uid, err := jwtutil.Parse(strings.TrimPrefix(header, prefix), secret)
	if err != nil {
		return 0, false
	}
	return uid, true
}

// Auth 校验 Authorization: Bearer <token>，通过后将 uid 注入上下文。
func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid, ok := parseUID(c, secret)
		if !ok {
			c.AbortWithStatusJSON(401, gin.H{"code": response.CodeUnauthorized, "msg": "未登录或凭证无效", "data": nil})
			return
		}
		c.Set(ctxUID, uid)
		c.Next()
	}
}

// AdminAuth 在 Auth 基础上校验数据库中的管理员角色（实时生效，改角色无需重新登录）。
func AdminAuth(secret string, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid, ok := parseUID(c, secret)
		if !ok {
			c.AbortWithStatusJSON(401, gin.H{"code": response.CodeUnauthorized, "msg": "未登录或凭证无效", "data": nil})
			return
		}
		var u model.User
		if err := db.Select("id", "role", "status").First(&u, uid).Error; err != nil || u.Role != "admin" {
			c.AbortWithStatusJSON(403, gin.H{"code": response.CodeForbidden, "msg": "需要管理员权限", "data": nil})
			return
		}
		if u.Status != 1 {
			c.AbortWithStatusJSON(403, gin.H{"code": response.CodeForbidden, "msg": "账号已被禁用", "data": nil})
			return
		}
		c.Set(ctxUID, uid)
		c.Next()
	}
}

// UID 取出鉴权中间件注入的用户 ID。
func UID(c *gin.Context) int64 {
	if v, ok := c.Get(ctxUID); ok {
		if id, ok := v.(int64); ok {
			return id
		}
	}
	return 0
}
