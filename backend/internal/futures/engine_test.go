package futures

import (
	"context"
	"testing"
	"time"

	"cryptosim/internal/model"

	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// setup 内存 SQLite + 注入固定价格的合约服务。初始 10,000 USDT。
func setup(t *testing.T) (*Service, *decimal.Decimal, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{}, &model.Balance{}, &model.LedgerEntry{},
		&model.FuturesPosition{}, &model.FundingRecord{},
	); err != nil {
		t.Fatalf("建表失败: %v", err)
	}
	if err := db.Create(&model.Balance{
		UserID: 1, Currency: "USDT", Available: decimal.NewFromInt(10000),
	}).Error; err != nil {
		t.Fatalf("初始化余额失败: %v", err)
	}
	price := decimal.NewFromInt(100)
	fn := func(ctx context.Context, symbol string) (decimal.Decimal, error) { return price, nil }
	return NewService(db, fn), &price, db
}

func usdt(t *testing.T, db *gorm.DB) decimal.Decimal {
	t.Helper()
	var b model.Balance
	if err := db.Where("user_id = 1 AND currency = 'USDT'").First(&b).Error; err != nil {
		t.Fatalf("查询余额失败: %v", err)
	}
	return b.Available
}

func TestOpenLongMarginAndFee(t *testing.T) {
	svc, _, db := setup(t)
	// 开多 0.5 BTC @100，5x：名义 50，保证金 10，手续费 0.025
	pos, err := svc.Open(context.Background(), 1, "BTCUSDT", SideLong, 5, decimal.NewFromFloat(0.5))
	if err != nil {
		t.Fatalf("开仓失败: %v", err)
	}
	if !pos.Margin.Equal(decimal.NewFromFloat(10)) {
		t.Fatalf("保证金 = %s, want 10", pos.Margin)
	}
	want := decimal.NewFromFloat(10000 - 10 - 0.025)
	if !usdt(t, db).Equal(want) {
		t.Fatalf("USDT = %s, want %s", usdt(t, db), want)
	}
}

func TestLiquidationLong(t *testing.T) {
	svc, price, db := setup(t)
	// 10x 做多：强平距离约 -9.5%，价格跌到 90 必然触发
	if _, err := svc.Open(context.Background(), 1, "BTCUSDT", SideLong, 10, decimal.NewFromFloat(1)); err != nil {
		t.Fatalf("开仓失败: %v", err)
	}
	*price = decimal.NewFromInt(90)
	n, err := svc.Liquidate(context.Background())
	if err != nil {
		t.Fatalf("强平检查失败: %v", err)
	}
	if n != 1 {
		t.Fatalf("应强平 1 个仓位, got %d", n)
	}
	var p model.FuturesPosition
	db.First(&p)
	if p.Status != "liquidated" {
		t.Fatalf("状态 = %s, want liquidated", p.Status)
	}
	if !p.RealizedPnl.IsNegative() {
		t.Fatalf("强平已实现盈亏应为负, got %s", p.RealizedPnl)
	}
	// 爆仓后权益归零：USDT 不变（开仓时已扣）
	if !usdt(t, db).Equal(decimal.NewFromFloat(10000 - 10 - 0.05)) {
		t.Fatalf("爆仓后 USDT 异常: %s", usdt(t, db))
	}
}

func TestLiquidationShort(t *testing.T) {
	svc, price, db := setup(t)
	// 10x 做空：价格涨到 110 触发强平（+9.5%）
	if _, err := svc.Open(context.Background(), 1, "BTCUSDT", SideShort, 10, decimal.NewFromFloat(1)); err != nil {
		t.Fatalf("开仓失败: %v", err)
	}
	*price = decimal.NewFromInt(110)
	n, _ := svc.Liquidate(context.Background())
	if n != 1 {
		t.Fatalf("空单应被强平, got %d", n)
	}
	var p model.FuturesPosition
	db.First(&p)
	if p.Status != "liquidated" {
		t.Fatalf("状态 = %s", p.Status)
	}
}

func TestNoLiquidationWhenHealthy(t *testing.T) {
	svc, price, db := setup(t)
	if _, err := svc.Open(context.Background(), 1, "BTCUSDT", SideLong, 10, decimal.NewFromFloat(1)); err != nil {
		t.Fatalf("开仓失败: %v", err)
	}
	*price = decimal.NewFromInt(95) // 跌 5%，未到 -9.5% 强平线
	n, _ := svc.Liquidate(context.Background())
	if n != 0 {
		t.Fatalf("健康仓位不应被强平, got %d", n)
	}
	var cnt int64
	db.Model(&model.FuturesPosition{}).Where("status = ?", "open").Count(&cnt)
	if cnt != 1 {
		t.Fatalf("持仓应仍为 open")
	}
}

func TestFundingSettlement(t *testing.T) {
	svc, _, db := setup(t)
	// 多头与空头各一个仓位，名义价值相同 → 多头付、空头收
	if _, err := svc.Open(context.Background(), 1, "BTCUSDT", SideLong, 5, decimal.NewFromFloat(1)); err != nil {
		t.Fatalf("开多失败: %v", err)
	}
	if _, err := svc.Open(context.Background(), 1, "ETHUSDT", SideShort, 5, decimal.NewFromFloat(10)); err != nil {
		t.Fatalf("开空失败: %v", err)
	}
	var long, short model.FuturesPosition
	db.Where("side = ?", SideLong).First(&long)
	db.Where("side = ?", SideShort).First(&short)
	longMargin, shortMargin := long.Margin, short.Margin

	// 把 last_funding_at 拨回 9 小时前，触发到期结算
	db.Model(&long).Update("last_funding_at", time.Now().Add(-9*time.Hour))
	db.Model(&short).Update("last_funding_at", time.Now().Add(-9*time.Hour))

	if _, err := svc.SettleFunding(context.Background()); err != nil {
		t.Fatalf("资金费率结算失败: %v", err)
	}
	var afterLong, afterShort model.FuturesPosition
	db.First(&afterLong, long.ID)
	db.First(&afterShort, short.ID)
	if afterLong.Margin.Cmp(longMargin) >= 0 {
		t.Fatalf("多头保证金应减少（付费率）: %s -> %s", longMargin, afterLong.Margin)
	}
	if afterShort.Margin.Cmp(shortMargin) <= 0 {
		t.Fatalf("空头保证金应增加（收费率）: %s -> %s", shortMargin, afterShort.Margin)
	}
	var records int64
	db.Model(&model.FundingRecord{}).Count(&records)
	if records != 2 {
		t.Fatalf("资金费率记录 = %d, want 2", records)
	}
}

func TestCloseLongProfit(t *testing.T) {
	svc, price, db := setup(t)
	if _, err := svc.Open(context.Background(), 1, "BTCUSDT", SideLong, 5, decimal.NewFromFloat(1)); err != nil {
		t.Fatalf("开仓失败: %v", err)
	}
	*price = decimal.NewFromInt(110) // 涨 10%
	if err := svc.Close(context.Background(), 1, 1, decimal.Zero); err != nil {
		t.Fatalf("平仓失败: %v", err)
	}
	var p model.FuturesPosition
	db.First(&p)
	if p.Status != "closed" {
		t.Fatalf("状态 = %s, want closed", p.Status)
	}
	// 已实现盈亏 = (110-100)×1 = 10（未扣平仓手续费前的方向正确）
	if !p.RealizedPnl.Equal(decimal.NewFromInt(10)) {
		t.Fatalf("已实现盈亏 = %s, want 10", p.RealizedPnl)
	}
	// 到账 = 保证金 20 + 盈亏 10 - 平仓手续费 0.055；开仓已扣 20 + 0.05
	want := decimal.NewFromFloat(10000 - 20 - 0.05 + 20 + 10 - 0.055)
	if got := usdt(t, db); !got.Equal(want) {
		t.Fatalf("USDT = %s, want %s", got, want)
	}
}

func TestOpenMinNotional(t *testing.T) {
	svc, _, _ := setup(t)
	// 名义价值 100×0.01 = 1 USDT < 5 → 拒绝
	_, err := svc.Open(context.Background(), 1, "BTCUSDT", SideLong, 5, decimal.NewFromFloat(0.01))
	if err == nil {
		t.Fatal("最小名义价值应拒绝")
	}
}
