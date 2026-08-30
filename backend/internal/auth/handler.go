package auth

import (
	"errors"
	"net/http"
	"time"

	"cryptosim/internal/middleware"
	"cryptosim/internal/pkg/jwtutil"
	"cryptosim/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc       *Service
	jwtSecret string
	jwtTTL    time.Duration
}

func NewHandler(svc *Service, secret string, ttl time.Duration) *Handler {
	return &Handler{svc: svc, jwtSecret: secret, jwtTTL: ttl}
}

type registerReq struct {
	Email    string `json:"email" binding:"required"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type loginReq struct {
	Account  string `json:"account" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// authResp 统一返回 token + HMAC 凭证对（apiSecret 明文仅出现这一次）。
type authResp struct {
	Token     string      `json:"token"`
	ApiKey    string      `json:"api_key"`
	ApiSecret string      `json:"api_secret"`
	User      interface{} `json:"user"`
}

func (h *Handler) Register(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "参数缺失")
		return
	}
	u, apiKey, apiSecret, err := h.svc.Register(
		RegisterInput{Email: req.Email, Username: req.Username, Password: req.Password},
		c.ClientIP(), c.Request.UserAgent(),
	)
	if err != nil {
		respondBizErr(c, err)
		return
	}
	token, err := jwtutil.Generate(u.ID, h.jwtSecret, h.jwtTTL)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "签发凭证失败")
		return
	}
	response.OK(c, authResp{Token: token, ApiKey: apiKey, ApiSecret: apiSecret, User: u})
}

func (h *Handler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "参数缺失")
		return
	}
	u, apiKey, apiSecret, err := h.svc.Login(req.Account, req.Password, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		respondBizErr(c, err)
		return
	}
	token, err := jwtutil.Generate(u.ID, h.jwtSecret, h.jwtTTL)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "签发凭证失败")
		return
	}
	response.OK(c, authResp{Token: token, ApiKey: apiKey, ApiSecret: apiSecret, User: u})
}

// ResetCredentials POST /auth/credentials/reset —— 轮换 HMAC 凭证（需登录）。
func (h *Handler) ResetCredentials(c *gin.Context) {
	apiKey, apiSecret, err := h.svc.ResetCredentials(middleware.UID(c))
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "重置失败")
		return
	}
	response.OK(c, gin.H{"api_key": apiKey, "api_secret": apiSecret})
}

// respondBizErr 将业务错误映射为统一错误码。
func respondBizErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrBadEmail), errors.Is(err, ErrBadUsername), errors.Is(err, ErrBadPassword):
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
	case errors.Is(err, ErrEmailTaken), errors.Is(err, ErrUsernameTaken):
		response.Fail(c, http.StatusBadRequest, response.CodeAccountTaken, err.Error())
	case errors.Is(err, ErrBadCredentials):
		response.Fail(c, http.StatusUnauthorized, response.CodeBadCredentials, err.Error())
	case errors.Is(err, ErrDisabled):
		response.Fail(c, http.StatusForbidden, response.CodeInvalidParams, err.Error())
	default:
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误")
	}
}
