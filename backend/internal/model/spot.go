package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// SpotOrder 现货订单。limit 单以限价成交（模拟对手盘按挂单价吃单），
// 市价单按实时价 ± 滑点立即成交。
type SpotOrder struct {
	ID          int64           `gorm:"primaryKey" json:"id"`
	UserID      int64           `json:"user_id"`
	Symbol      string          `json:"symbol"`
	Side        string          `json:"side"` // buy / sell
	Type        string          `json:"type"` // limit / market
	Price       decimal.Decimal `gorm:"type:numeric(38,18)" json:"price"`
	Amount      decimal.Decimal `gorm:"type:numeric(38,18)" json:"amount"`
	Filled      decimal.Decimal `gorm:"type:numeric(38,18)" json:"filled"`
	AvgPrice    decimal.Decimal `gorm:"type:numeric(38,18)" json:"avg_price"`
	Fee         decimal.Decimal `gorm:"type:numeric(38,18)" json:"fee"`
	FrozenQuote decimal.Decimal `gorm:"type:numeric(38,18)" json:"-"` // 买单剩余冻结 USDT
	ClientOrderID string        `gorm:"column:client_order_id" json:"-"` // 客户端幂等号
	TriggerPrice  decimal.Decimal `gorm:"type:numeric(38,18)" json:"trigger_price"` // 条件单触发价（>0 为条件单）
	PostOnly      bool            `gorm:"column:post_only" json:"post_only"`
	Status      string          `json:"status"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

func (SpotOrder) TableName() string { return "spot_orders" }

// Trade 成交记录。
type Trade struct {
	ID          int64           `gorm:"primaryKey" json:"id"`
	OrderID     int64           `json:"order_id"`
	UserID      int64           `json:"user_id"`
	Symbol      string          `json:"symbol"`
	Side        string          `json:"side"`
	Price       decimal.Decimal `gorm:"type:numeric(38,18)" json:"price"`
	Amount      decimal.Decimal `gorm:"type:numeric(38,18)" json:"amount"`
	QuoteAmount decimal.Decimal `gorm:"type:numeric(38,18)" json:"quote_amount"`
	Fee         decimal.Decimal `gorm:"type:numeric(38,18)" json:"fee"`
	CreatedAt   time.Time       `json:"created_at"`
}

func (Trade) TableName() string { return "trades" }
