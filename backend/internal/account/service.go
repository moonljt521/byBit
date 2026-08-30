// Package account 账户查询与重置。
package account

import (
	"cryptosim/internal/ledger"
	"cryptosim/internal/model"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Service struct {
	db          *gorm.DB
	initialUSDT decimal.Decimal
}

func NewService(db *gorm.DB, initialUSDT string) *Service {
	return &Service{db: db, initialUSDT: decimal.RequireFromString(initialUSDT)}
}

func (s *Service) Me(uid int64) (*model.User, []model.Balance, error) {
	var u model.User
	if err := s.db.First(&u, uid).Error; err != nil {
		return nil, nil, err
	}
	var balances []model.Balance
	if err := s.db.Where("user_id = ?", uid).Order("currency").Find(&balances).Error; err != nil {
		return nil, nil, err
	}
	return &u, balances, nil
}

// Reset 将账户恢复为初始状态：清空全部余额，重新赠送初始虚拟 USDT。
// 历史流水保留（审计可追溯），重置动作本身也记一笔流水。
func (s *Service) Reset(uid int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", uid).Delete(&model.Balance{}).Error; err != nil {
			return err
		}
		return ledger.GrantUSDT(tx, uid, s.initialUSDT, "reset_grant", "", "重置账户，恢复初始虚拟资金")
	})
}
