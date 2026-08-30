// Package futures USDT 本位永续合约（逐仓）：市价开/平仓、资金费率结算、强平。
// 仿真规则：taker 0.05%；维持保证金率 0.5%；资金费率每 8 小时结算 0.01%（多头付空头）；
// 强平条件：保证金 + 未实现盈亏 ≤ 维持保证金，强平将损失全部剩余保证金（近似真实爆仓）。
package futures

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
	ErrInvalidSide     = errors.New("side 取值 long/short")
	ErrInvalidLeverage = errors.New("杠杆取值 1-20")
	ErrInvalidAmount   = errors.New("数量必须大于 0")
	ErrTooSmall        = errors.New("开仓金额太小（最小 5 USDT 名义价值）")
	ErrPositionNotFound = errors.New("仓位不存在")
	ErrPositionNotOpen  = errors.New("仓位已结束")
	ErrNoPrice          = errors.New("暂时获取不到行情价格，请稍后再试")
	ErrCloseTooLarge    = errors.New("平仓数量超过持仓")
	ErrMarketNoPrice = errors.New("行情不可用")
)

const (
	feeTakerF     = "0.0005"    // taker 0.05%
	maintRateF    = "0.005"     // 维持保证金率 0.5%
	fundingRateF  = "0.0001"    // 每 8 小时 0.01%
	minNotionalF  = "5"
	fundingEvery  = 8 * time.Hour
	SideLong      = "long"
	SideShort     = "short"
)

type Service struct {
	db    *gorm.DB
	lastPriceFn func(ctx context.Context, symbol string) (decimal.Decimal, error)
	// Notify 强平事件回调（由 server 注入 WebSocket Hub）
	Notify     func(userID int64, event any)
	fee        decimal.Decimal
	maint      decimal.Decimal
	funding    decimal.Decimal
	minNotion  decimal.Decimal
}

// PriceFn 实时价格函数（生产由 market.Service.LastPrice 提供，测试可注入）。
type PriceFn = func(ctx context.Context, symbol string) (decimal.Decimal, error)

func NewService(db *gorm.DB, lastPrice PriceFn) *Service {
	return &Service{
		db: db, lastPriceFn: lastPrice,
		fee:       decimal.RequireFromString(feeTakerF),
		maint:     decimal.RequireFromString(maintRateF),
		funding:   decimal.RequireFromString(fundingRateF),
		minNotion: decimal.RequireFromString(minNotionalF),
	}
}

func (s *Service) LastPrice(ctx context.Context, symbol string) (decimal.Decimal, error) {
	return s.lastPriceFn(ctx, symbol)
}

// Open 市价开仓。
func (s *Service) Open(ctx context.Context, uid int64, symbol, side string, leverage int, amount decimal.Decimal) (*model.FuturesPosition, error) {
	if side != SideLong && side != SideShort {
		return nil, ErrInvalidSide
	}
	if leverage < 1 || leverage > 20 {
		return nil, ErrInvalidLeverage
	}
	if amount.Cmp(decimal.Zero) <= 0 {
		return nil, ErrInvalidAmount
	}
	px, err := s.LastPrice(ctx, symbol)
	if err != nil {
		return nil, ErrNoPrice
	}
	notional := px.Mul(amount)
	if notional.Cmp(s.minNotion) < 0 {
		return nil, ErrTooSmall
	}
	margin := notional.Div(decimal.NewFromInt(int64(leverage))).Round(8)
	fee := notional.Mul(s.fee).Round(8)
	pos := &model.FuturesPosition{
		UserID: uid, Symbol: symbol, Side: side, Leverage: leverage,
		Size: amount, EntryPrice: px, Margin: margin, Status: "open",
		Fee: fee,
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := balance.Debit(tx, uid, "USDT", margin.Add(fee), "futures_margin", "合约开仓保证金+手续费"); err != nil {
			return err
		}
		return tx.Create(pos).Error
	})
	if err != nil {
		return nil, err
	}
	return pos, nil
}

// Close 平仓（amount 为空或 ≥ 剩余仓位时全部平掉）。
func (s *Service) Close(ctx context.Context, uid, positionID int64, amount decimal.Decimal) error {
	if amount.IsNegative() {
		return ErrInvalidAmount
	}
	var p model.FuturesPosition
	if err := s.db.Select("id", "symbol").First(&p, positionID).Error; err != nil {
		return ErrPositionNotFound
	}
	px, err := s.LastPrice(ctx, p.Symbol)
	if err != nil {
		return ErrNoPrice
	}
	return s.closeAt(uid, positionID, amount, px, "closed", "合约平仓")
}

func (s *Service) closeAt(uid, positionID int64, amount decimal.Decimal, mark decimal.Decimal, status string, memo string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var p model.FuturesPosition
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ?", positionID, uid).First(&p).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPositionNotFound
			}
			return err
		}
		if p.Status != "open" {
			return ErrPositionNotOpen
		}
		closeAmt := amount
		if closeAmt.IsZero() || closeAmt.Cmp(p.Size) >= 0 {
			closeAmt = p.Size
		}
		if closeAmt.Cmp(p.Size) > 0 {
			return ErrCloseTooLarge
		}
		dir := int64(1)
		if p.Side == SideShort {
			dir = -1
		}
		ratio := closeAmt.Div(p.Size)
		released := p.Margin.Mul(ratio).Round(8)
		pnl := mark.Sub(p.EntryPrice).Mul(closeAmt).Mul(decimal.NewFromInt(dir)).Round(8)
		fee := mark.Mul(closeAmt).Mul(s.fee).Round(8)
		settle := released.Add(pnl).Sub(fee)
		if settle.IsNegative() {
			settle = decimal.Zero
		}
		if err := balance.Credit(tx, uid, "USDT", settle, "futures_close", memo); err != nil {
			return err
		}
		newSize := p.Size.Sub(closeAmt)
		updates := map[string]any{
			"size":         newSize,
			"margin":       p.Margin.Sub(released),
			"realized_pnl": p.RealizedPnl.Add(pnl),
			"fee":          p.Fee.Add(fee),
			"updated_at":   time.Now(),
		}
		if newSize.IsZero() {
			now := time.Now()
			updates["status"] = status
			updates["closed_at"] = &now
		}
		if err := tx.Model(&p).Updates(updates).Error; err != nil {
			return err
		}
		return nil
	})
}

// LiquidationPrice 估算强平价（展示用）。
func (s *Service) LiquidationPrice(p *model.FuturesPosition) decimal.Decimal {
	lev := decimal.NewFromInt(int64(p.Leverage))
	one := decimal.NewFromInt(1)
	if p.Side == SideLong {
		return p.EntryPrice.Mul(one.Sub(one.Div(lev)).Add(s.maint)).Round(2)
	}
	return p.EntryPrice.Mul(one.Add(one.Div(lev)).Sub(s.maint)).Round(2)
}

// Liquidate 强平：保证金 + 未实现盈亏 ≤ 维持保证金时触发，损失全部剩余保证金。
func (s *Service) Liquidate(ctx context.Context) (int, error) {
	var positions []model.FuturesPosition
	if err := s.db.Where("status = ?", "open").Find(&positions).Error; err != nil {
		return 0, err
	}
	count := 0
	for i := range positions {
		p := positions[i]
		mark, err := s.LastPrice(ctx, p.Symbol)
		if err != nil {
			continue
		}
		dir := int64(1)
		if p.Side == SideShort {
			dir = -1
		}
		uPnL := mark.Sub(p.EntryPrice).Mul(p.Size).Mul(decimal.NewFromInt(dir))
		equity := p.Margin.Add(uPnL)
		maint := mark.Mul(p.Size).Mul(s.maint)
		if equity.Cmp(maint) > 0 {
			continue
		}
		// 爆仓：损失全部保证金（近似真实逐仓强平，剩余价值归保险基金）
		err = s.db.Transaction(func(tx *gorm.DB) error {
			var cur model.FuturesPosition
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&cur, p.ID).Error; err != nil {
				return err
			}
			if cur.Status != "open" {
				return nil
			}
			now := time.Now()
			return tx.Model(&cur).Updates(map[string]any{
				"status": "liquidated", "closed_at": &now,
				"realized_pnl": cur.Margin.Neg(), "size": decimal.Zero,
				"updated_at": now,
			}).Error
		})
		if err == nil {
			count++
		}
	}
	return count, nil
}

// SettleFunding 资金费率结算：多头向空头支付（模拟固定 0.01%/8h）。
func (s *Service) SettleFunding(ctx context.Context) (int, error) {
	due := time.Now().Add(-fundingEvery)
	var positions []model.FuturesPosition
	if err := s.db.Where("status = ? AND last_funding_at <= ?", "open", due).Find(&positions).Error; err != nil {
		return 0, err
	}
	count := 0
	for i := range positions {
		p := positions[i]
		mark, err := s.LastPrice(ctx, p.Symbol)
		if err != nil {
			continue
		}
		amount := mark.Mul(p.Size).Mul(s.funding).Round(8) // 正数；多头支付、空头收取
		err = s.db.Transaction(func(tx *gorm.DB) error {
			var cur model.FuturesPosition
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&cur, p.ID).Error; err != nil {
				return err
			}
			if cur.Status != "open" {
				return nil
			}
			newMargin := cur.Margin
			record := model.FundingRecord{
				PositionID: cur.ID, UserID: cur.UserID, Symbol: cur.Symbol,
				Rate: s.funding, Amount: amount.Neg(),
			}
			if cur.Side == SideLong {
				newMargin = cur.Margin.Sub(amount)
			} else {
				newMargin = cur.Margin.Add(amount)
				record.Amount = amount
			}
			now := time.Now()
			if err := tx.Model(&cur).Updates(map[string]any{
				"margin": newMargin, "last_funding_at": now, "updated_at": now,
			}).Error; err != nil {
				return err
			}
			return tx.Create(&record).Error
		})
		if err == nil {
			count++
		}
	}
	return count, nil
}

// OpenPositions 当前持仓。
func (s *Service) OpenPositions(uid int64) ([]model.FuturesPosition, error) {
	var out []model.FuturesPosition
	err := s.db.Where("user_id = ? AND status = ?", uid, "open").Order("id DESC").Find(&out).Error
	return out, err
}

// History 历史仓位。
func (s *Service) History(uid int64) ([]model.FuturesPosition, error) {
	var out []model.FuturesPosition
	err := s.db.Where("user_id = ? AND status <> ?", uid, "open").Order("id DESC").Limit(100).Find(&out).Error
	return out, err
}

// FundingHistory 资金费率记录。
func (s *Service) FundingHistory(uid int64) ([]model.FundingRecord, error) {
	var out []model.FundingRecord
	err := s.db.Where("user_id = ?", uid).Order("id DESC").Limit(100).Find(&out).Error
	return out, err
}
