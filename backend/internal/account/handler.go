package account

import (
	"errors"
	"net/http"

	"cryptosim/internal/middleware"
	"cryptosim/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Me(c *gin.Context) {
	u, balances, err := h.svc.Me(middleware.UID(c))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Unauthorized(c, "用户不存在")
			return
		}
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误")
		return
	}
	response.OK(c, gin.H{"user": u, "balances": balances})
}

func (h *Handler) Reset(c *gin.Context) {
	if err := h.svc.Reset(middleware.UID(c)); err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "重置失败")
		return
	}
	response.OK(c, nil)
}
