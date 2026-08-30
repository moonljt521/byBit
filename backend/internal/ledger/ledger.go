// Package ledger 资金账本操作。所有余额变动必须经由本包产生流水，
// 保证「任何一笔钱都能查出来龙去脉」——真实交易所账务系统的基本要求。
package ledger

import (
	"errors"
	"time"

	"cryptosim/internal/model"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// GrantUSDT 在事务内为用户入账 USDT（余额不存在则创建），
// 并写入对应流水。必须在外层 gorm 事务中调用。
// M3 接入撮合后，下单冻结/成交划转等将在此包扩展（含行锁与余额校验）。
func GrantUSDT(tx *gorm.DB, userID int64, amount decimal.Decimal, bizType, bizID, memo string) error {
	var bal model.Balance
	err := tx.Where("user_id = ? AND currency = ?", userID, "USDT").First(&bal).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		bal = model.Balance{UserID: userID, Currency: "USDT", Available: amount}
		if err := tx.Create(&bal).Error; err != nil {
			return err
		}
	case err != nil:
		return err
	default:
		bal.Available = bal.Available.Add(amount)
		if err := tx.Model(&bal).Updates(map[string]any{
			"available":  bal.Available,
			"updated_at": time.Now(),
		}).Error; err != nil {
			return err
		}
	}
	entry := model.LedgerEntry{
		UserID:       userID,
		BizType:      bizType,
		BizID:        bizID,
		Currency:     "USDT",
		Amount:       amount,
		BalanceAfter: bal.Available,
		Memo:         memo,
	}
	return tx.Create(&entry).Error
}
