package market

import (
	"net/http"
	"strconv"

	"cryptosim/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Tickers GET /market/tickers —— 现货 24h 行情列表（公开接口）。
func (h *Handler) Tickers(c *gin.Context) {
	v, err := h.svc.Tickers(c.Request.Context())
	if err != nil {
		response.Fail(c, http.StatusBadGateway, response.CodeInternal, "行情源不可用")
		return
	}
	response.OK(c, gin.H{"symbols": h.svc.Symbols(), "tickers": v})
}

// Coins GET /market/coins —— 币种目录。
func (h *Handler) Coins(c *gin.Context) {
	v, err := h.svc.Coins()
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误")
		return
	}
	response.OK(c, v)
}

// Klines GET /market/klines?symbol=BTCUSDT&bar=1m&limit=200
func (h *Handler) Klines(c *gin.Context) {
	symbol := sanitizeSymbol(c.Query("symbol"))
	if !ValidSymbol(symbol) {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "交易对格式不正确")
		return
	}
	bar := c.DefaultQuery("bar", "1m")
	if !Bars[bar] {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "不支持的K线周期")
		return
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "200"))
	if err != nil || limit < 1 || limit > 500 {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "limit 取值 1-500")
		return
	}
	v, err := h.svc.Klines(c.Request.Context(), symbol, bar, limit)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, response.CodeInternal, "行情源不可用")
		return
	}
	response.OK(c, gin.H{"symbol": symbol, "bar": bar, "candles": v})
}

// Depth GET /market/depth?symbol=BTCUSDT&size=15
func (h *Handler) Depth(c *gin.Context) {
	symbol := sanitizeSymbol(c.Query("symbol"))
	if !ValidSymbol(symbol) {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "交易对格式不正确")
		return
	}
	size, err := strconv.Atoi(c.DefaultQuery("size", "15"))
	if err != nil || size < 1 || size > 50 {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "size 取值 1-50")
		return
	}
	v, err := h.svc.Depth(c.Request.Context(), symbol, size)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, response.CodeInternal, "行情源不可用")
		return
	}
	response.OK(c, v)
}

// Trades GET /market/trades?symbol=BTCUSDT&limit=20
func (h *Handler) Trades(c *gin.Context) {
	symbol := sanitizeSymbol(c.Query("symbol"))
	if !ValidSymbol(symbol) {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "交易对格式不正确")
		return
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil || limit < 1 || limit > 100 {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "limit 取值 1-100")
		return
	}
	v, err := h.svc.Trades(c.Request.Context(), symbol, limit)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, response.CodeInternal, "行情源不可用")
		return
	}
	response.OK(c, v)
}
