package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimit 基于 IP 的内存滑动窗口限流（单实例部署够用，多实例应改用 Redis）。
// 每个 RateLimit() 实例拥有独立的计数存储。
func RateLimit(max int, window time.Duration) gin.HandlerFunc {
	var mu sync.Mutex
	store := map[string][]time.Time{}
	return func(c *gin.Context) {
		key := c.ClientIP()
		now := time.Now()
		mu.Lock()
		arr := store[key]
		fresh := arr[:0]
		for _, t := range arr {
			if now.Sub(t) < window {
				fresh = append(fresh, t)
			}
		}
		if len(fresh) >= max {
			store[key] = fresh
			mu.Unlock()
			c.AbortWithStatusJSON(http.StatusTooManyRequests,
				gin.H{"code": 10429, "msg": "请求过于频繁，请稍后再试", "data": nil})
			return
		}
		store[key] = append(fresh, now)
		mu.Unlock()
		c.Next()
	}
}

// SecureHeaders 商业标配安全响应头。
func SecureHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("X-Permitted-Cross-Domain-Policies", "none")
		// 生产启用 HTTPS 后由 nginx 追加 HSTS
		c.Next()
	}
}
