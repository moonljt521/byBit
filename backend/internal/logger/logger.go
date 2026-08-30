// Package logger 全局 zap 日志：dev 用控制台彩色输出，production 用 JSON。
package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var global *zap.Logger

// Init 按环境初始化全局 logger。
func Init(env string) {
	var cfg zap.Config
	if env == "production" {
		cfg = zap.NewProductionConfig()
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	global = zap.Must(cfg.Build())
	zap.ReplaceGlobals(global)
}

// L 返回全局 logger（未初始化时退回 zap 示例，便于测试）。
func L() *zap.Logger {
	if global == nil {
		global = zap.Must(zap.NewDevelopment())
	}
	return global
}

func Sync() {
	if global != nil {
		_ = global.Sync()
	}
}
