package futures

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"cryptosim/internal/balance"
	"cryptosim/internal/market"
	"cryptosim/internal/middleware"
	"cryptosim/internal/model"
	"cryptosim/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

type openReq struct {
	Symbol   string `json:"symbol" binding:"required"`
	Side     string `json:"side" binding:"required"`
	Leverage int    `json:"leverage" binding:"required"`
	Amount   string `json:"amount" binding:"required"`
}

// Open POST /futures/positions
func (h *Handler) Open(c *gin.Context) {
	var req openReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "参数缺失")
		return
	}
	symbol := strings.ToUpper(strings.TrimSpace(req.Symbol))
	if !market.ValidSymbol(symbol) {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "交易对格式不正确")
		return
	}
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "数量格式不正确")
		return
	}
	pos, err := h.svc.Open(c.Request.Context(), middleware.UID(c), symbol, req.Side, req.Leverage, amount)
	if err != nil {
		switch {
		case errors.Is(err, balance.ErrInsufficient):
			response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "可用 USDT 不足（需保证金+手续费）")
		default:
			response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
		}
		return
	}
	response.OK(c, pos)
}

type closeReq struct {
	Amount string `json:"amount"`
}

// Close POST /futures/positions/:id/close
func (h *Handler) Close(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "仓位 ID 不合法")
		return
	}
	amount := decimal.Zero
	var req closeReq
	if err := c.ShouldBindJSON(&req); err == nil && req.Amount != "" {
		amount, err = decimal.NewFromString(req.Amount)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "数量格式不正确")
			return
		}
	}
	if err := h.svc.Close(c.Request.Context(), middleware.UID(c), id, amount); err != nil {
		switch {
		case errors.Is(err, ErrPositionNotFound):
			response.Fail(c, http.StatusNotFound, response.CodeInvalidParams, err.Error())
		case errors.Is(err, ErrNoPrice):
			response.Fail(c, http.StatusServiceUnavailable, response.CodeInternal, err.Error())
		default:
			response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
		}
		return
	}
	response.OK(c, nil)
}

type PositionView struct {
	model.FuturesPosition
	MarkPrice        decimal.Decimal `json:"mark_price"`
	UnrealizedPnl    decimal.Decimal `json:"unrealized_pnl"`
	ROI              string          `json:"roi"`
	LiquidationPrice decimal.Decimal `json:"liquidation_price"`
}

func (h *Handler) view(ctx context.Context, p model.FuturesPosition) PositionView {
	mark, err := h.svc.LastPrice(ctx, p.Symbol)
	if err != nil {
		mark = p.EntryPrice
	}
	dir := int64(1)
	if p.Side == SideShort {
		dir = -1
	}
	uPnL := mark.Sub(p.EntryPrice).Mul(p.Size).Mul(decimal.NewFromInt(dir)).Round(8)
	roi := "0"
	if p.Margin.IsPositive() {
		roi = uPnL.Div(p.Margin).Mul(decimal.NewFromInt(100)).Round(2).StringFixed(2)
	}
	return PositionView{
		FuturesPosition:  p,
		MarkPrice:        mark,
		UnrealizedPnl:    uPnL,
		ROI:              roi,
		LiquidationPrice: h.svc.LiquidationPrice(&p),
	}
}

// Positions GET /futures/positions
func (h *Handler) Positions(c *gin.Context) {
	list, err := h.svc.OpenPositions(middleware.UID(c))
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误")
		return
	}
	out := make([]PositionView, 0, len(list))
	for _, p := range list {
		out = append(out, h.view(c.Request.Context(), p))
	}
	response.OK(c, out)
}

// History GET /futures/positions/history
func (h *Handler) History(c *gin.Context) {
	list, err := h.svc.History(middleware.UID(c))
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误")
		return
	}
	out := make([]PositionView, 0, len(list))
	for _, p := range list {
		out = append(out, h.view(c.Request.Context(), p))
	}
	response.OK(c, out)
}

// Funding GET /futures/funding
func (h *Handler) Funding(c *gin.Context) {
	list, err := h.svc.FundingHistory(middleware.UID(c))
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误")
		return
	}
	response.OK(c, list)
}
