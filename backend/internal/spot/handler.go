package spot

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"cryptosim/internal/balance"
	"cryptosim/internal/middleware"
	"cryptosim/internal/market"
	"cryptosim/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

type placeReq struct {
	Symbol        string `json:"symbol" binding:"required"`
	Side          string `json:"side" binding:"required"`
	Type          string `json:"type" binding:"required"`
	Price         string `json:"price"`
	Amount        string `json:"amount" binding:"required"`
	ClientOrderID string `json:"client_order_id"`
	TriggerPrice  string `json:"trigger_price"`
	PostOnly      bool   `json:"post_only"`
}

// Place POST /spot/orders
func (h *Handler) Place(c *gin.Context) {
	var req placeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "参数缺失")
		return
	}
	symbol := strings.ToUpper(strings.TrimSpace(req.Symbol))
	if !market.ValidSymbol(symbol) {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "交易对格式不正确")
		return
	}
	price := decimal.Zero
	if req.Price != "" {
		p, err := decimal.NewFromString(req.Price)
		if err != nil || p.IsNegative() {
			response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "价格格式不正确")
			return
		}
		price = p
	}
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "数量格式不正确")
		return
	}
	trigger := decimal.Zero
	if req.TriggerPrice != "" {
		tp, err := decimal.NewFromString(req.TriggerPrice)
		if err != nil || tp.IsNegative() {
			response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "触发价格式不正确")
			return
		}
		trigger = tp
	}
	order, err := h.svc.Place(c.Request.Context(), middleware.UID(c), PlaceInput{
		Symbol: symbol, Side: req.Side, Type: req.Type, Price: price, Amount: amount,
		ClientOrderID: strings.TrimSpace(req.ClientOrderID),
		TriggerPrice:  trigger, PostOnly: req.PostOnly,
	})
	if err != nil {
		switch {
		case errors.Is(err, balance.ErrInsufficient):
			response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "余额不足")
		case errors.Is(err, ErrPostOnlyMatch), errors.Is(err, ErrTriggerInvalid):
			response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
		case errors.Is(err, ErrMarketNoPrice):
			response.Fail(c, http.StatusServiceUnavailable, response.CodeInternal, err.Error())
		default:
			response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
		}
		return
	}
	response.OK(c, order)
}

// Cancel DELETE /spot/orders/:id
func (h *Handler) Cancel(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "订单 ID 不合法")
		return
	}
	if err := h.svc.Cancel(middleware.UID(c), id); err != nil {
		switch {
		case errors.Is(err, ErrOrderNotFound):
			response.Fail(c, http.StatusNotFound, response.CodeInvalidParams, err.Error())
		case errors.Is(err, ErrOrderNotOpen):
			response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
		default:
			response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误")
		}
		return
	}
	response.OK(c, nil)
}

// OpenOrders GET /spot/orders/open
func (h *Handler) OpenOrders(c *gin.Context) {
	out, err := h.svc.OpenOrders(middleware.UID(c))
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误")
		return
	}
	response.OK(c, out)
}

// History GET /spot/orders/history
func (h *Handler) History(c *gin.Context) {
	out, err := h.svc.History(middleware.UID(c))
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误")
		return
	}
	response.OK(c, out)
}

// MyTrades GET /spot/trades
func (h *Handler) MyTrades(c *gin.Context) {
	out, err := h.svc.MyTrades(middleware.UID(c))
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误")
		return
	}
	response.OK(c, out)
}
