// Package zaperr 提供 zap.Field 别名，避免业务代码直接依赖 zap 细节。
package zaperr

import "go.uber.org/zap"

// Err 是 zap.Error 字段的短别名。
func Err(err error) zap.Field { return zap.Error(err) }
