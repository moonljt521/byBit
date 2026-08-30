// Package spot 现货交易：下单 / 撤单 / 查询 + 撮合引擎。
// 模拟盘撮合模型（与行业模拟盘一致）：
//   - 限价单挂入本方订单簿，当真实市场价穿越限价时，由「流动性引擎」分批成交（价格-时间优先、部分成交）；
//   - 市价单按实时价 ± 滑点立即全额成交；
//   - 买单冻结 USDT（价格×数量×(1+费率)），卖单冻结 base 币，撤单解冻。
package spot

import (
	"context"
	"errors"
	"time"

	"cryptosim/internal/balance"
	"cryptosim/internal/model"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrInvalidSide    = errors.New("side 取值 buy/sell")
	ErrInvalidType    = errors.New("type 取值 limit/market")
	ErrInvalidPrice   = errors.New("限价单价格必须大于 0")
	ErrInvalidAmount  = errors.New("数量必须大于 0")
	ErrOrderNotFound  = errors.New("订单不存在")
	ErrOrderNotOpen   = errors.New("订单已结束，无法撤销")
	ErrMarketNoPrice  = errors.New("暂时获取不到行情价格，请稍后再试")
	ErrAmountTooSmall = errors.New("订单金额太小（最小 5 USDT）")
	ErrPostOnlyMatch  = errors.New("Post-Only 限价单会立即成交，已拒绝")
	ErrTriggerInvalid = errors.New("触发价必须大于 0")
)

const (
	feeTaker   = "0.001"  // taker 手续费 0.1%
	slippage   = "0.0005" // 市价单滑点 0.05%
	minAmount  = "0.00000001"
	minNotional = "5" // 最小下单金额（USDT），与真实交易所一致
	SideBuy    = "buy"
	SideSell   = "sell"
	TypeLimit  = "limit"
	TypeMarket = "market"
)

type Service struct {
	db          *gorm.DB
	lastPriceFn PriceFn
	// Notify 成交事件回调（由 server 注入 WebSocket Hub，向用户推送成交通知）
	Notify      func(userID int64, event any)
	fee         decimal.Decimal
	slip        decimal.Decimal
	minAmount   decimal.Decimal
	minNotional decimal.Decimal
}

// PriceFn 实时价格函数（生产由 market.Service.LastPrice 提供，测试可注入假价格）。
type PriceFn = func(ctx context.Context, symbol string) (decimal.Decimal, error)

func NewService(db *gorm.DB, lastPrice PriceFn) *Service {
	return &Service{
		db:          db,
		lastPriceFn: lastPrice,
		fee:         decimal.RequireFromString(feeTaker),
		slip:        decimal.RequireFromString(slippage),
		minAmount:   decimal.RequireFromString(minAmount),
		minNotional: decimal.RequireFromString(minNotional),
	}
}

type PlaceInput struct {
	Symbol        string
	Side          string
	Type          string
	Price         decimal.Decimal
	Amount        decimal.Decimal
	ClientOrderID string          // 幂等号：同用户重复提交返回已有订单
	TriggerPrice  decimal.Decimal // >0 为条件单：buy 在市场价>=trigger 触发，sell 在市场价<=trigger 触发
	PostOnly      bool            // 限价单若会立即成交则拒绝（保证 maker）
}

// LastPrice 实时最新价。
func (s *Service) LastPrice(ctx context.Context, symbol string) (decimal.Decimal, error) {
	return s.lastPriceFn(ctx, symbol)
}

// Place 下单。市价单在事务内立即成交；限价单挂单等待引擎撮合。
func (s *Service) Place(ctx context.Context, uid int64, in PlaceInput) (*model.SpotOrder, error) {
	if in.Side != SideBuy && in.Side != SideSell {
		return nil, ErrInvalidSide
	}
	if in.Type != TypeLimit && in.Type != TypeMarket {
		return nil, ErrInvalidType
	}
	if in.Amount.Cmp(s.minAmount) <= 0 {
		return nil, ErrInvalidAmount
	}
	if in.Type == TypeLimit && in.TriggerPrice.IsZero() {
		if in.Price.Cmp(decimal.Zero) <= 0 {
			return nil, ErrInvalidPrice
		}
		if in.Price.Mul(in.Amount).Cmp(s.minNotional) < 0 {
			return nil, ErrAmountTooSmall
		}
	}
	if in.TriggerPrice.IsPositive() && in.TriggerPrice.Mul(in.Amount).Cmp(s.minNotional) < 0 {
		return nil, ErrAmountTooSmall
	}

	// 幂等：重复提交同一 client_order_id 返回原订单
	if o, ok := s.findExisting(uid, in.ClientOrderID); ok {
		return o, nil
	}

	if in.TriggerPrice.IsPositive() {
		if in.Type != TypeLimit {
			return nil, ErrTriggerInvalid
		}
		if o, ok := s.findExisting(uid, in.ClientOrderID); ok {
			return o, nil
		}
		return s.placeTrigger(uid, in)
	}
	if in.Type == TypeMarket {
		return s.placeMarket(ctx, uid, in)
	}
	if in.PostOnly {
		px, err := s.LastPrice(ctx, in.Symbol)
		if err != nil {
			return nil, ErrMarketNoPrice
		}
		if (in.Side == SideBuy && px.LessThan(in.Price)) ||
			(in.Side == SideSell && px.GreaterThan(in.Price)) {
			return nil, ErrPostOnlyMatch
		}
	}
	return s.placeLimit(uid, in)
}

// findExisting 幂等查询：同用户同 client_order_id 的历史订单直接返回。
func (s *Service) findExisting(uid int64, clientOrderID string) (*model.SpotOrder, bool) {
	if clientOrderID == "" {
		return nil, false
	}
	var o model.SpotOrder
	err := s.db.Where("user_id = ? AND client_order_id = ?", uid, clientOrderID).First(&o).Error
	if err != nil {
		return nil, false
	}
	return &o, true
}

// placeTrigger 条件单：挂单等待触发，触发后由引擎按市价 ± 滑点成交。
func (s *Service) placeTrigger(uid int64, in PlaceInput) (*model.SpotOrder, error) {
	return s.placeLimit(uid, in)
}

func (s *Service) placeLimit(uid int64, in PlaceInput) (*model.SpotOrder, error) {
	one := decimal.NewFromInt(1)
	feeFactor := one.Add(s.fee)
	order := &model.SpotOrder{
		UserID: uid, Symbol: in.Symbol, Side: in.Side, Type: TypeLimit,
		Price: in.Price, Amount: in.Amount, Status: "pending",
		ClientOrderID: in.ClientOrderID, PostOnly: in.PostOnly,
	}
	freezePrice := in.Price
	if in.TriggerPrice.IsPositive() { // 条件单冻结按触发价计
		order.TriggerPrice = in.TriggerPrice
		freezePrice = in.TriggerPrice
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if in.Side == SideBuy {
			frozen := freezePrice.Mul(in.Amount).Mul(feeFactor)
			if err := balance.Freeze(tx, uid, "USDT", frozen); err != nil {
				return err
			}
			order.FrozenQuote = frozen
		} else {
			if err := balance.Freeze(tx, uid, baseOf(in.Symbol), in.Amount); err != nil {
				return err
			}
		}
		return tx.Create(order).Error
	})
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (s *Service) placeMarket(ctx context.Context, uid int64, in PlaceInput) (*model.SpotOrder, error) {
	px, err := s.LastPrice(ctx, in.Symbol)
	if err != nil {
		return nil, ErrMarketNoPrice
	}
	if px.Mul(in.Amount).Cmp(s.minNotional) < 0 {
		return nil, ErrAmountTooSmall
	}
	one := decimal.NewFromInt(1)
	var fillPrice decimal.Decimal
	if in.Side == SideBuy {
		fillPrice = px.Mul(one.Add(s.slip))
	} else {
		fillPrice = px.Mul(one.Sub(s.slip))
	}
	order := &model.SpotOrder{
		UserID: uid, Symbol: in.Symbol, Side: in.Side, Type: TypeMarket,
		Amount: in.Amount, Status: "filled",
		ClientOrderID: in.ClientOrderID,
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		quote := fillPrice.Mul(in.Amount)
		var fee decimal.Decimal
		if in.Side == SideBuy {
			fee = quote.Mul(s.fee)
			if err := balance.Debit(tx, uid, "USDT", quote.Add(fee), "trade", "市价买入"); err != nil {
				return err
			}
			if err := balance.Credit(tx, uid, baseOf(in.Symbol), in.Amount, "trade", "市价买入成交"); err != nil {
				return err
			}
		} else {
			fee = quote.Mul(s.fee)
			if err := balance.Debit(tx, uid, baseOf(in.Symbol), in.Amount, "trade", "市价卖出"); err != nil {
				return err
			}
			if err := balance.Credit(tx, uid, "USDT", quote.Sub(fee), "trade", "市价卖出成交"); err != nil {
				return err
			}
		}
		order.Price = fillPrice
		order.Filled = in.Amount
		order.AvgPrice = fillPrice
		order.Fee = fee
		if err := tx.Create(order).Error; err != nil {
			return err
		}
		return tx.Create(&model.Trade{
			OrderID: order.ID, UserID: uid, Symbol: in.Symbol, Side: in.Side,
			Price: fillPrice, Amount: in.Amount, QuoteAmount: quote, Fee: fee,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return order, nil
}

// Cancel 撤销挂单并解冻剩余资金。
func (s *Service) Cancel(uid, orderID int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var o model.SpotOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ?", orderID, uid).First(&o).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrOrderNotFound
			}
			return err
		}
		if o.Status != "pending" && o.Status != "partial" {
			return ErrOrderNotOpen
		}
		if o.Side == SideBuy {
			if err := balance.Unfreeze(tx, uid, "USDT", o.FrozenQuote); err != nil {
				return err
			}
			o.FrozenQuote = decimal.Zero
		} else {
			remain := o.Amount.Sub(o.Filled)
			if err := balance.Unfreeze(tx, uid, baseOf(o.Symbol), remain); err != nil {
				return err
			}
		}
		o.Status = "canceled"
		o.UpdatedAt = time.Now()
		return tx.Model(&o).Updates(map[string]any{
			"status": "canceled", "frozen_quote": decimal.Zero, "updated_at": time.Now(),
		}).Error
	})
}

func (s *Service) OpenOrders(uid int64) ([]model.SpotOrder, error) {
	var out []model.SpotOrder
	err := s.db.Where("user_id = ? AND status IN ?", uid, []string{"pending", "partial"}).
		Order("id DESC").Limit(100).Find(&out).Error
	return out, err
}

func (s *Service) History(uid int64) ([]model.SpotOrder, error) {
	var out []model.SpotOrder
	err := s.db.Where("user_id = ? AND status IN ?", uid, []string{"filled", "canceled"}).
		Order("id DESC").Limit(100).Find(&out).Error
	return out, err
}

func (s *Service) MyTrades(uid int64) ([]model.Trade, error) {
	var out []model.Trade
	err := s.db.Where("user_id = ?", uid).Order("id DESC").Limit(100).Find(&out).Error
	return out, err
}

func baseOf(symbol string) string {
	return trimQuote(symbol)
}

// PendingOrders 供撮合引擎取待成交订单。
func (s *Service) PendingOrders(limit int) ([]model.SpotOrder, error) {
	var out []model.SpotOrder
	err := s.db.Where("status IN ? AND type = ?", []string{"pending", "partial"}, TypeLimit).
		Order("id").Limit(limit).Find(&out).Error
	return out, err
}

func trimQuote(symbol string) string {
	for i := len(symbol) - 4; i >= 0; i-- {
		if symbol[i:] == "USDT" {
			return symbol[:i]
		}
	}
	return symbol
}
