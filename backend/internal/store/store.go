// Package store 负责数据库 / Redis 连接、启动迁移与种子数据。
// Redis 在 M1 为可选依赖：连不上仅告警，不影响账户功能，M2 行情模块强依赖。
package store

import (
	"context"
	"encoding/json"
	"time"

	"cryptosim/internal/config"
	"cryptosim/internal/logger"
	"cryptosim/internal/model"
	"cryptosim/internal/pkg/zaperr"
	"cryptosim/migrations"

	"github.com/golang-migrate/migrate/v4"
	migpg "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/redis/go-redis/v9"
	gormpg "gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type Store struct {
	DB    *gorm.DB
	Redis *redis.Client
}

func New(ctx context.Context, cfg *config.Config) (*Store, error) {
	db, err := gorm.Open(gormpg.Open(cfg.DBDSN), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetConnMaxLifetime(time.Hour)

	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.L().Warn("redis 不可用，M1 功能不受影响（M2 行情模块需要）", zaperr.Err(err))
		rdb = nil
	}

	return &Store{DB: db, Redis: rdb}, nil
}

// Migrate 执行内嵌 SQL 迁移到最新版本。
func (s *Store) Migrate() error {
	sqlDB, err := s.DB.DB()
	if err != nil {
		return err
	}
	driver, err := migpg.WithInstance(sqlDB, &migpg.Config{})
	if err != nil {
		return err
	}
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return err
	}
	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}

// Seed 幂等写入币种与交易对种子数据。
func (s *Store) Seed() error {
	coins := []model.Coin{
		{Symbol: "BTC", DisplayName: "Bitcoin 比特币", Sort: 1, Enabled: true},
		{Symbol: "ETH", DisplayName: "Ethereum 以太坊", Sort: 2, Enabled: true},
		{Symbol: "USDT", DisplayName: "Tether 泰达币", Sort: 3, Enabled: true},
		{Symbol: "BNB", DisplayName: "BNB 币安币", Sort: 4, Enabled: true},
		{Symbol: "SOL", DisplayName: "Solana", Sort: 5, Enabled: true},
		{Symbol: "XRP", DisplayName: "XRP 瑞波币", Sort: 6, Enabled: true},
		{Symbol: "TRX", DisplayName: "TRON 波场", Sort: 7, Enabled: true},
		{Symbol: "DOGE", DisplayName: "Dogecoin 狗狗币", Sort: 8, Enabled: true},
	}
	for i := range coins {
		if coins[i].Meta == nil {
			coins[i].Meta = json.RawMessage("{}")
		}
		if err := s.DB.Where(model.Coin{Symbol: coins[i].Symbol}).
			FirstOrCreate(&coins[i]).Error; err != nil {
			return err
		}
	}
	pairs := []model.TradingPair{
		{Symbol: "BTCUSDT", BaseCurrency: "BTC", QuoteCurrency: "USDT", PairType: "spot", Enabled: true},
		{Symbol: "ETHUSDT", BaseCurrency: "ETH", QuoteCurrency: "USDT", PairType: "spot", Enabled: true},
		{Symbol: "BNBUSDT", BaseCurrency: "BNB", QuoteCurrency: "USDT", PairType: "spot", Enabled: true},
		{Symbol: "SOLUSDT", BaseCurrency: "SOL", QuoteCurrency: "USDT", PairType: "spot", Enabled: true},
		{Symbol: "XRPUSDT", BaseCurrency: "XRP", QuoteCurrency: "USDT", PairType: "spot", Enabled: true},
		{Symbol: "TRXUSDT", BaseCurrency: "TRX", QuoteCurrency: "USDT", PairType: "spot", Enabled: true},
		{Symbol: "DOGEUSDT", BaseCurrency: "DOGE", QuoteCurrency: "USDT", PairType: "spot", Enabled: true},
		{Symbol: "BTCUSDT", BaseCurrency: "BTC", QuoteCurrency: "USDT", PairType: "futures", Enabled: true},
		{Symbol: "ETHUSDT", BaseCurrency: "ETH", QuoteCurrency: "USDT", PairType: "futures", Enabled: true},
	}
	for i := range pairs {
		if err := s.DB.Where(model.TradingPair{Symbol: pairs[i].Symbol, PairType: pairs[i].PairType}).
			FirstOrCreate(&pairs[i]).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) Close() {
	if sqlDB, err := s.DB.DB(); err == nil {
		_ = sqlDB.Close()
	}
	if s.Redis != nil {
		_ = s.Redis.Close()
	}
}
