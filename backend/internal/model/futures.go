package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// FuturesPosition 永续合约仓位（逐仓）。
type FuturesPosition struct {
	ID            int64           `gorm:"primaryKey" json:"id"`
	UserID        int64           `json:"user_id"`
	Symbol        string          `json:"symbol"`
	Side          string          `json:"side"` // long / short
	Leverage      int             `json:"leverage"`
	Size          decimal.Decimal `gorm:"type:numeric(38,18)" json:"size"`
	EntryPrice    decimal.Decimal `gorm:"type:numeric(38,18)" json:"entry_price"`
	Margin        decimal.Decimal `gorm:"type:numeric(38,18)" json:"margin"`
	Status        string          `json:"status"` // open / closed / liquidated
	RealizedPnl   decimal.Decimal `gorm:"type:numeric(38,18)" json:"realized_pnl"`
	Fee           decimal.Decimal `gorm:"type:numeric(38,18)" json:"fee"`
	LastFundingAt time.Time       `json:"last_funding_at"`
	OpenedAt      time.Time       `json:"opened_at"`
	ClosedAt      *time.Time      `json:"closed_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

func (FuturesPosition) TableName() string { return "futures_positions" }

// FundingRecord 资金费率结算记录。
type FundingRecord struct {
	ID         int64           `gorm:"primaryKey" json:"id"`
	PositionID int64           `json:"position_id"`
	UserID     int64           `json:"user_id"`
	Symbol     string          `json:"symbol"`
	Rate       decimal.Decimal `gorm:"type:numeric(38,18)" json:"rate"`
	Amount     decimal.Decimal `gorm:"type:numeric(38,18)" json:"amount"` // 正=收（空），负=付（多）
	CreatedAt  time.Time       `json:"created_at"`
}

func (FundingRecord) TableName() string { return "funding_records" }
