// Package response 统一 HTTP 响应封装：{"code":0,"msg":"ok","data":...}。
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 业务错误码
const (
	CodeOK             = 0
	CodeInvalidParams  = 10001
	CodeAccountTaken   = 10002
	CodeBadCredentials = 10003
	CodeUnauthorized   = 10401
	CodeForbidden      = 10403
	CodeInternal       = 10500
)

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"code": CodeOK, "msg": "ok", "data": data})
}

func Fail(c *gin.Context, httpStatus, code int, msg string) {
	c.JSON(httpStatus, gin.H{"code": code, "msg": msg, "data": nil})
}

func Unauthorized(c *gin.Context, msg string) {
	Fail(c, http.StatusUnauthorized, CodeUnauthorized, msg)
}
