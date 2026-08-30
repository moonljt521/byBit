package spot

import (
	"context"
	"testing"

	"cryptosim/internal/balance"
	"cryptosim/internal/model"

	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// benchSvc 内存库压测环境：初始 1,000,000 USDT，价格固定 100。
func benchSvc(b *testing.B) (*Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		b.Fatal(err)
	}
	if err := db.AutoMigrate(
		&model.User{}, &model.Balance{}, &model.LedgerEntry{},
		&model.SpotOrder{}, &model.Trade{},
	); err != nil {
		b.Fatal(err)
	}
	if err := db.Create(&model.Balance{
		UserID: 1, Currency: "USDT", Available: decimal.NewFromInt(1000000),
	}).Error; err != nil {
		b.Fatal(err)
	}
	fn := func(ctx context.Context, symbol string) (decimal.Decimal, error) {
		return decimal.NewFromInt(100), nil
	}
	return NewService(db, fn), db
}

func benchSvc2(b *testing.B) (*Service, *gorm.DB, *decimal.Decimal) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		b.Fatal(err)
	}
	if err := db.AutoMigrate(
		&model.User{}, &model.Balance{}, &model.LedgerEntry{},
		&model.SpotOrder{}, &model.Trade{},
	); err != nil {
		b.Fatal(err)
	}
	if err := db.Create(&model.Balance{
		UserID: 1, Currency: "USDT", Available: decimal.NewFromInt(1000000),
	}).Error; err != nil {
		b.Fatal(err)
	}
	price := decimal.NewFromInt(50) // 与挂单价相同 → 不触发穿越，纯扫描
	fn := func(ctx context.Context, symbol string) (decimal.Decimal, error) { return price, nil }
	return NewService(db, fn), db, &price
}

// BenchmarkPlaceLimit 下单全链路（验签后的业务层：校验→冻结→落库→流水）。
func BenchmarkPlaceLimit(b *testing.B) {
	svc, _ := benchSvc(b)
	ctx := context.Background()
	in := PlaceInput{
		Symbol: "BTCUSDT", Side: SideBuy, Type: TypeLimit,
		Price: decimal.NewFromInt(50), Amount: decimal.NewFromFloat(0.2), // 10 USDT 名义价值，永不成交
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := svc.Place(ctx, 1, in); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEngineScanPending 引擎扫描 500 条挂单（无成交路径）。
func BenchmarkEngineScanPending(b *testing.B) {
	svc, _, _ := benchSvc2(b)
	ctx := context.Background()
	in := PlaceInput{
		Symbol: "BTCUSDT", Side: SideBuy, Type: TypeLimit,
		Price: decimal.NewFromInt(50), Amount: decimal.NewFromFloat(0.2),
	}
	for i := 0; i < 500; i++ {
		if _, err := svc.Place(ctx, 1, in); err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		orders, err := svc.PendingOrders(500)
		if err != nil {
			b.Fatal(err)
		}
		_ = orders
	}
}

// BenchmarkBalanceFreezeCredit 资金冻结+入账+流水（复式记账写路径）。
func BenchmarkBalanceFreezeCredit(b *testing.B) {
	_, db := benchSvc(b)
	amt := decimal.NewFromFloat(0.001)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := db.Transaction(func(tx *gorm.DB) error {
			if err := balance.Freeze(tx, 1, "USDT", amt); err != nil {
				return err
			}
			return balance.Credit(tx, 1, "BTC", amt, "trade", "bench")
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}
