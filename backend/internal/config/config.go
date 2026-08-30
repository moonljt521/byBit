// Package config 从环境变量加载配置，全部带本地开发默认值，生产环境用 SIM_* 覆盖。
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env            string        // dev / production
	HTTPAddr       string        // 监听地址
	DBDSN          string        // PostgreSQL 连接串
	RedisAddr      string        // Redis 地址，M1 可选
	JWTSecret      string        // JWT 签名密钥
	JWTExpire      time.Duration // token 有效期
	InitialUSDT    string        // 注册赠送的虚拟 USDT（decimal 字符串）
	HTTPProxy      string        // 上游行情请求代理（如 http://127.0.0.1:26002），空则走 HTTPS_PROXY 环境变量或直连
	LearnDir       string        // 学习中心内容目录
	EncKey         string        // 敏感字段落库加密主密钥（生产必须更换）
	AuthRateLimit  int           // 登录/注册限流：次数 / 分钟 / IP
	AllowedOrigins []string      // CORS 白名单
}

func Load() *Config {
	return &Config{
		Env:            getEnv("SIM_ENV", "dev"),
		HTTPAddr:       getEnv("SIM_HTTP_ADDR", ":8080"),
		DBDSN:          getEnv("SIM_DB_DSN", "postgres://cryptosim:cryptosim@127.0.0.1:5432/cryptosim?sslmode=disable"),
		RedisAddr:      getEnv("SIM_REDIS_ADDR", "127.0.0.1:6379"),
		JWTSecret:      getEnv("SIM_JWT_SECRET", "dev-secret-do-not-use-in-production"),
		JWTExpire:      24 * time.Hour,
		InitialUSDT:    getEnv("SIM_INITIAL_USDT", "10000"),
		HTTPProxy:      getEnv("SIM_HTTP_PROXY", ""),
		LearnDir:       getEnv("SIM_LEARN_DIR", "../content/learning"),
		EncKey:         getEnv("SIM_ENC_KEY", "dev-enc-key-change-in-production"),
		AuthRateLimit:  getEnvInt("SIM_AUTH_RATE_LIMIT", 20),		AllowedOrigins: strings.Split(getEnv("SIM_ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173,http://localhost:5174,http://127.0.0.1:5174"), ","),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}
