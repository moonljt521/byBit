package model

import (
	"encoding/json"
	"time"
)

// Coin 币种目录，meta 预留给学习中心（简介、共识机制、总量等）。
type Coin struct {
	ID          int64           `gorm:"primaryKey" json:"id"`
	Symbol      string          `json:"symbol"`
	DisplayName string          `json:"display_name"`
	Sort        int             `json:"sort"`
	Enabled     bool            `json:"enabled"`
	Meta        json.RawMessage `gorm:"type:jsonb" json:"meta"`
	CreatedAt   time.Time       `json:"created_at"`
}

func (Coin) TableName() string { return "coins" }

// TradingPair 交易对，现货与合约共用一张表，pair_type 区分。
type TradingPair struct {
	ID            int64     `gorm:"primaryKey" json:"id"`
	Symbol        string    `json:"symbol"` // BTCUSDT
	BaseCurrency  string    `json:"base_currency"`
	QuoteCurrency string    `json:"quote_currency"`
	PairType      string    `json:"pair_type"` // spot / futures
	Enabled       bool      `json:"enabled"`
	CreatedAt     time.Time `json:"created_at"`
}

func (TradingPair) TableName() string { return "trading_pairs" }
