package admin

import (
	"errors"
	"net/http"
	"strconv"

	"cryptosim/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Stats(c *gin.Context) {
	st, err := h.svc.Stats()
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误")
		return
	}
	response.OK(c, st)
}

func pageParams(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	return page, size
}

func (h *Handler) Users(c *gin.Context) {
	page, size := pageParams(c)
	rows, total, err := h.svc.Users(page, size, c.Query("keyword"))
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误")
		return
	}
	response.OK(c, gin.H{"list": rows, "total": total, "page": page, "page_size": size})
}

type statusReq struct {
	Status *int16 `json:"status" binding:"required"`
}

func (h *Handler) SetStatus(c *gin.Context) {
	uid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "用户 ID 不合法")
		return
	}
	var req statusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "参数缺失")
		return
	}
	if err := h.svc.SetStatus(uid, *req.Status); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, http.StatusNotFound, response.CodeInvalidParams, "用户不存在")
			return
		}
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
		return
	}
	response.OK(c, nil)
}

type grantReq struct {
	Amount string `json:"amount" binding:"required"`
	Memo   string `json:"memo"`
}

func (h *Handler) AdjustFunds(c *gin.Context) {
	uid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "用户 ID 不合法")
		return
	}
	var req grantReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "参数缺失")
		return
	}
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "金额格式不正确")
		return
	}
	if err := h.svc.AdjustFunds(uid, amount, req.Memo); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *Handler) Ledger(c *gin.Context) {
	page, size := pageParams(c)
	uid, _ := strconv.ParseInt(c.Query("user_id"), 10, 64)
	rows, total, err := h.svc.Ledger(page, size, uid)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误")
		return
	}
	response.OK(c, gin.H{"list": rows, "total": total, "page": page, "page_size": size})
}

// LoginLogs GET /admin/login-logs
func (h *Handler) LoginLogs(c *gin.Context) {
	page, size := pageParams(c)
	rows, total, err := h.svc.LoginLogs(page, size)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误")
		return
	}
	response.OK(c, gin.H{"list": rows, "total": total, "page": page, "page_size": size})
}
