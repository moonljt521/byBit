// Package migrations 内嵌 SQL 迁移文件，随二进制发布，启动时由 golang-migrate 执行。
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
