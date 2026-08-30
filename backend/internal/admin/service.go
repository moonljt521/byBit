// Package admin 后台管理：统计看板、用户管理、资金调拨、流水审计。
package admin

import (
	"errors"
	"time"

	"cryptosim/internal/ledger"
	"cryptosim/internal/model"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

type Stats struct {
	TotalUsers    int64           `json:"total_users"`
	NewUsersToday int64           `json:"new_users_today"`
	ActiveUsers   int64           `json:"active_users"` // status=1
	USDTAvailable decimal.Decimal `json:"usdt_available"`
	USDTCold      decimal.Decimal `json:"usdt_frozen"`
	LedgerToday   int64           `json:"ledger_today"`
}

func (s *Service) Stats() (*Stats, error) {
	var st Stats
	today := time.Now().Format("2006-01-02")
	if err := s.db.Model(&model.User{}).Count(&st.TotalUsers).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&model.User{}).Where("created_at >= ?", today).Count(&st.NewUsersToday).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&model.User{}).Where("status = 1").Count(&st.ActiveUsers).Error; err != nil {
		return nil, err
	}
	type sumRow struct {
		Available decimal.Decimal
		Frozen    decimal.Decimal
	}
	var sum sumRow
	if err := s.db.Model(&model.Balance{}).
		Select("COALESCE(SUM(available),0) AS available, COALESCE(SUM(frozen),0) AS frozen").
		Where("currency = ?", "USDT").Scan(&sum).Error; err != nil {
		return nil, err
	}
	st.USDTAvailable = sum.Available
	st.USDTCold = sum.Frozen
	if err := s.db.Model(&model.LedgerEntry{}).Where("created_at >= ?", today).Count(&st.LedgerToday).Error; err != nil {
		return nil, err
	}
	return &st, nil
}

type UserRow struct {
	model.User
	USDTAvailable decimal.Decimal `json:"usdt_available"`
}

// Users 分页用户列表（keyword 匹配邮箱/用户名），附带 USDT 可用余额。
func (s *Service) Users(page, pageSize int, keyword string) ([]UserRow, int64, error) {
	query := s.db.Model(&model.User{})
	if keyword != "" {
		kw := "%" + keyword + "%"
		query = query.Where("email ILIKE ? OR username ILIKE ?", kw, kw)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []model.User
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]UserRow, 0, len(users))
	for _, u := range users {
		var bal model.Balance
		err := s.db.Where("user_id = ? AND currency = ?", u.ID, "USDT").First(&bal).Error
		avail := decimal.Zero
		if err == nil {
			avail = bal.Available
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, err
		}
		rows = append(rows, UserRow{User: u, USDTAvailable: avail})
	}
	return rows, total, nil
}

// SetStatus 启用/禁用用户。
func (s *Service) SetStatus(uid int64, status int16) error {
	if status != 0 && status != 1 {
		return errors.New("status 取值 0 或 1")
	}
	res := s.db.Model(&model.User{}).Where("id = ?", uid).Updates(map[string]any{"status": status, "updated_at": time.Now()})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// AdjustFunds 管理员调拨虚拟资金：amount 可正可负（负数扣减，余额不足报错），记入流水。
func (s *Service) AdjustFunds(uid int64, amount decimal.Decimal, memo string) error {
	if amount.IsZero() {
		return errors.New("金额不能为 0")
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if amount.IsNegative() {
			var bal model.Balance
			err := tx.Where("user_id = ? AND currency = ?", uid, "USDT").First(&bal).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("该用户无 USDT 余额")
			}
			if err != nil {
				return err
			}
			if bal.Available.Add(amount).IsNegative() {
				return errors.New("调减后余额不能为负")
			}
		}
		return ledger.GrantUSDT(tx, uid, amount, "admin_adjust", "", "管理员调拨: "+memo)
	})
}

type LedgerRow struct {
	model.LedgerEntry
	Username string `json:"username"`
}

// LoginLogs 登录审计日志（分页）。
func (s *Service) LoginLogs(page, pageSize int) ([]model.LoginLog, int64, error) {
	var total int64
	if err := s.db.Model(&model.LoginLog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.LoginLog
	err := s.db.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error
	return rows, total, err
}

// Ledger 分页流水（可按用户过滤）。
func (s *Service) Ledger(page, pageSize int, uid int64) ([]LedgerRow, int64, error) {
	query := s.db.Model(&model.LedgerEntry{})
	if uid > 0 {
		query = query.Where("user_id = ?", uid)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []LedgerRow
	err := query.
		Select("ledger_entries.*, users.username AS username").
		Joins("JOIN users ON users.id = ledger_entries.user_id").
		Order("ledger_entries.id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Scan(&rows).Error
	return rows, total, err
}
