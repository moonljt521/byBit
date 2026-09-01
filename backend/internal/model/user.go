// Package model GORM 数据模型，字段与 migrations/0001_init.up.sql 一一对应。
package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type User struct {
	ID           int64     `gorm:"primaryKey" json:"id"`
	Email        string    `json:"email"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         string    `gorm:"default:user" json:"role"` // user / admin
	Status       int16     `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (User) TableName() string { return "users" }

// Balance 用户某币种余额：available 可用，frozen 挂单冻结。
type Balance struct {
	ID        int64           `gorm:"primaryKey" json:"id"`
	UserID    int64           `json:"user_id"`
	Currency  string          `json:"currency"`
	Available decimal.Decimal `gorm:"type:numeric(38,18)" json:"available"`
	Frozen    decimal.Decimal `gorm:"type:numeric(38,18)" json:"frozen"`
	UpdatedAt time.Time       `json:"updated_at"`
}

func (Balance) TableName() string { return "balances" }

// LedgerEntry 资金流水：amount 正为入账、负为出账，balance_after 记录变动后可用余额。
type LedgerEntry struct {
	ID           int64           `gorm:"primaryKey" json:"id"`
	UserID       int64           `json:"user_id"`
	BizType      string          `json:"biz_type"`
	BizID        string          `json:"biz_id"`
	Currency     string          `json:"currency"`
	Amount       decimal.Decimal `gorm:"type:numeric(38,18)" json:"amount"`
	BalanceAfter decimal.Decimal `gorm:"type:numeric(38,18)" json:"balance_after"`
	Memo         string          `json:"memo"`
	CreatedAt    time.Time       `json:"created_at"`
}

func (LedgerEntry) TableName() string { return "ledger_entries" }
