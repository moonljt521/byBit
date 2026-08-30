package spot

import (
	"context"
	"math/rand"
	"time"

	"cryptosim/internal/balance"
	"cryptosim/internal/logger"
	"cryptosim/internal/model"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Engine 限价单撮合引擎：每秒扫描挂单，当真实市场价穿越限价时分批成交。
// 批量成交比例随机（40%-100%），模拟真实盘口的逐笔流动性。
type Engine struct {
	svc *Service
	rng *rand.Rand
}

func NewEngine(svc *Service) *Engine {
	return &Engine{svc: svc, rng: rand.New(rand.NewSource(time.Now().UnixNano()))}
}

func (e *Engine) slipPrice() decimal.Decimal { return e.svc.slip }

// Run 阻塞运行，ctx 取消后退出。
func (e *Engine) Run(ctx context.Context) {
	logger.L().Info("撮合引擎启动", zap.Duration("interval", time.Second))
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.L().Info("撮合引擎停止")
			return
		case <-t.C:
			e.tick(ctx)
		}
	}
}

func (e *Engine) tick(ctx context.Context) {
	orders, err := e.svc.PendingOrders(500)
	if err != nil {
		logger.L().Warn("撮合引擎读取挂单失败", zap.Error(err))
		return
	}
	for i := range orders {
		select {
		case <-ctx.Done():
			return
		default:
		}
		o := orders[i]
		px, err := e.svc.LastPrice(ctx, o.Symbol)
		if err != nil {
			continue // 行情暂时不可用，下个 tick 再试
		}
		// 条件单：buy 在市场价 >= 触发价时触发，sell 在市场价 <= 触发价时触发，按市价成交
		if o.TriggerPrice.IsPositive() {
			triggered := (o.Side == SideBuy && px.Cmp(o.TriggerPrice) >= 0) ||
				(o.Side == SideSell && px.Cmp(o.TriggerPrice) <= 0)
			if !triggered {
				continue
			}
			remaining := o.Amount.Sub(o.Filled)
			fillPx := px
			if o.Side == SideBuy {
				fillPx = px.Mul(decimal.NewFromInt(1).Add(e.slipPrice()))
			} else {
				fillPx = px.Mul(decimal.NewFromInt(1).Sub(e.slipPrice()))
			}
			if err := e.svc.fillOrder(o.ID, remaining, fillPx); err != nil {
				logger.L().Warn("条件单触发成交失败", zap.Int64("order", o.ID), zap.Error(err))
			}
			continue
		}

		// 价格穿越判定：买单在市场价 ≤ 限价时成交；卖单在市场价 ≥ 限价时成交
		if o.Side == SideBuy && px.Cmp(o.Price) > 0 {
			continue
		}
		if o.Side == SideSell && px.Cmp(o.Price) < 0 {
			continue
		}
		remaining := o.Amount.Sub(o.Filled)
		if remaining.Cmp(decimal.Zero) <= 0 {
			continue
		}
		// 尾单清扫：剩余价值不足 1 USDT 时一次吃完，避免无限长尾
		remainingValue := remaining.Mul(o.Price)
		if o.Side == SideSell {
			remainingValue = remaining.Mul(px)
		}
		if remainingValue.Cmp(decimal.NewFromInt(1)) < 0 {
			if err := e.svc.fillOrder(o.ID, remaining, decimal.Zero); err != nil {
				logger.L().Warn("尾单清扫失败", zap.Int64("order", o.ID), zap.Error(err))
			}
			continue
		}
		frac := 0.4 + e.rng.Float64()*0.6
		fillAmt := remaining.Mul(decimal.NewFromFloat(frac)).Round(8)
		if fillAmt.Cmp(decimal.Zero) <= 0 || fillAmt.Cmp(remaining) >= 0 {
			fillAmt = remaining
		}
		if err := e.svc.fillOrder(o.ID, fillAmt, decimal.Zero); err != nil {
			logger.L().Warn("撮合成交失败", zap.Int64("order", o.ID), zap.Error(err))
		}
	}
}

// fillOrder 成交一笔（事务 + 行锁防并发重复成交）。fillPrice 非零时以其为成交价（条件单按市价），否则按限价。
func (s *Service) fillOrder(orderID int64, fillAmt, fillPrice decimal.Decimal) error {
	one := decimal.NewFromInt(1)
	feeFactor := one.Add(s.fee)
	return s.db.Transaction(func(tx *gorm.DB) error {
		var o model.SpotOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&o, orderID).Error; err != nil {
			return err
		}
		if o.Status != "pending" && o.Status != "partial" {
			return nil // 已被撤销/成交完
		}
		remaining := o.Amount.Sub(o.Filled)
		if fillAmt.Cmp(remaining) > 0 {
			fillAmt = remaining
		}
		if fillPrice.IsZero() {
			fillPrice = o.Price // 模拟对手盘按挂单价吃单
		}
		quote := fillAmt.Mul(fillPrice)
		fee := quote.Mul(s.fee)
		base := baseOf(o.Symbol)
		final := o.Filled.Add(fillAmt).Cmp(o.Amount) >= 0

		if o.Side == SideBuy {
			consumed := quote.Mul(feeFactor)
			if err := balance.ConsumeFrozen(tx, o.UserID, "USDT", consumed); err != nil {
				return err
			}
			if final {
				// 全部成交：解冻舍入产生的剩余冻结
				residual := o.FrozenQuote.Sub(consumed)
				if residual.IsNegative() {
					residual = decimal.Zero
				}
				if residual.IsPositive() {
					if err := balance.Unfreeze(tx, o.UserID, "USDT", residual); err != nil {
						return err
					}
				}
				o.FrozenQuote = decimal.Zero
			} else {
				o.FrozenQuote = o.FrozenQuote.Sub(consumed)
			}
			if err := balance.Credit(tx, o.UserID, base, fillAmt, "trade", "限价买入成交"); err != nil {
				return err
			}
		} else {
			if err := balance.ConsumeFrozen(tx, o.UserID, base, fillAmt); err != nil {
				return err
			}
			if err := balance.Credit(tx, o.UserID, "USDT", quote.Sub(fee), "trade", "限价卖出成交"); err != nil {
				return err
			}
		}

		newFilled := o.Filled.Add(fillAmt)
		newAvg := o.AvgPrice.Mul(o.Filled).Add(fillPrice.Mul(fillAmt)).Div(newFilled)
		updates := map[string]any{
			"filled":     newFilled,
			"avg_price":  newAvg,
			"fee":        o.Fee.Add(fee),
			"updated_at": time.Now(),
		}
		status := "partial"
		if final {
			status = "filled"
		}
		updates["status"] = status
		updates["frozen_quote"] = o.FrozenQuote
		if err := tx.Model(&model.SpotOrder{}).Where("id = ?", o.ID).Updates(updates).Error; err != nil {
			return err
		}
		if err := tx.Create(&model.Trade{
			OrderID: o.ID, UserID: o.UserID, Symbol: o.Symbol, Side: o.Side,
			Price: fillPrice, Amount: fillAmt, QuoteAmount: quote, Fee: fee,
		}).Error; err != nil {
			return err
		}
		if s.Notify != nil {
			s.Notify(o.UserID, map[string]any{
				"type": "trade", "order_id": o.ID, "symbol": o.Symbol, "side": o.Side,
				"price": fillPrice.String(), "amount": fillAmt.String(), "fee": fee.String(),
			})
		}
		return nil
	})
}
