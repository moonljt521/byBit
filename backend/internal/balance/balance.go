// Package balance 余额原子操作：所有资金变动必须走这里（条件更新防超扣）+ 流水。
package balance

import (
	"errors"
	"time"

	"cryptosim/internal/model"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

var ErrInsufficient = errors.New("余额不足")

func writeLedger(tx *gorm.DB, userID int64, currency string, amount, balanceAfter decimal.Decimal, bizType, memo string) error {
	return tx.Create(&model.LedgerEntry{
		UserID: userID, BizType: bizType, Currency: currency,
		Amount: amount, BalanceAfter: balanceAfter, Memo: memo,
	}).Error
}

// Freeze 冻结：available → frozen（条件更新，防超扣）。
func Freeze(tx *gorm.DB, userID int64, currency string, amount decimal.Decimal) error {
	res := tx.Model(&model.Balance{}).
		Where("user_id = ? AND currency = ? AND available >= ?", userID, currency, amount).
		Updates(map[string]any{
			"available":  gorm.Expr("available - ?", amount),
			"frozen":     gorm.Expr("frozen + ?", amount),
			"updated_at": time.Now(),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrInsufficient
	}
	var bal model.Balance
	if err := tx.Where("user_id = ? AND currency = ?", userID, currency).First(&bal).Error; err != nil {
		return err
	}
	return writeLedger(tx, userID, currency, amount.Neg(), bal.Available, "order_freeze", "下单冻结")
}

// Unfreeze 解冻：frozen → available。
func Unfreeze(tx *gorm.DB, userID int64, currency string, amount decimal.Decimal) error {
	if amount.IsNegative() {
		amount = decimal.Zero
	}
	if amount.IsZero() {
		return nil
	}
	res := tx.Model(&model.Balance{}).
		Where("user_id = ? AND currency = ? AND frozen >= ?", userID, currency, amount).
		Updates(map[string]any{
			"available":  gorm.Expr("available + ?", amount),
			"frozen":     gorm.Expr("frozen - ?", amount),
			"updated_at": time.Now(),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrInsufficient
	}
	var bal model.Balance
	if err := tx.Where("user_id = ? AND currency = ?", userID, currency).First(&bal).Error; err != nil {
		return err
	}
	return writeLedger(tx, userID, currency, amount, bal.Available, "order_unfreeze", "撤单解冻")
}

// ConsumeFrozen 成交消耗冻结资金（frozen 直接扣减，不回到 available，无流水——
// 冻结时已记流水，成交的资产入账由调用方另行记流水）。
func ConsumeFrozen(tx *gorm.DB, userID int64, currency string, amount decimal.Decimal) error {
	if !amount.IsPositive() {
		return nil
	}
	res := tx.Model(&model.Balance{}).
		Where("user_id = ? AND currency = ? AND frozen >= ?", userID, currency, amount).
		Updates(map[string]any{
			"frozen":     gorm.Expr("frozen - ?", amount),
			"updated_at": time.Now(),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrInsufficient
	}
	return nil
}

// Credit 入账（余额行不存在则创建）。
func Credit(tx *gorm.DB, userID int64, currency string, amount decimal.Decimal, bizType, memo string) error {
	var bal model.Balance
	err := tx.Where("user_id = ? AND currency = ?", userID, currency).First(&bal).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		bal = model.Balance{UserID: userID, Currency: currency, Available: amount}
		if err := tx.Create(&bal).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		bal.Available = bal.Available.Add(amount)
		if err := tx.Model(&bal).Updates(map[string]any{
			"available":  bal.Available,
			"updated_at": time.Now(),
		}).Error; err != nil {
			return err
		}
	}
	return writeLedger(tx, userID, currency, amount, bal.Available, bizType, memo)
}

// Debit 扣减可用余额（条件更新防超扣）。
func Debit(tx *gorm.DB, userID int64, currency string, amount decimal.Decimal, bizType, memo string) error {
	res := tx.Model(&model.Balance{}).
		Where("user_id = ? AND currency = ? AND available >= ?", userID, currency, amount).
		Updates(map[string]any{
			"available":  gorm.Expr("available - ?", amount),
			"updated_at": time.Now(),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrInsufficient
	}
	var bal model.Balance
	if err := tx.Where("user_id = ? AND currency = ?", userID, currency).First(&bal).Error; err != nil {
		return err
	}
	return writeLedger(tx, userID, currency, amount.Neg(), bal.Available, bizType, memo)
}
