package market

import (
	"context"
	"encoding/json"
	"regexp"
	"strconv"
	"sync"
	"time"

	"cryptosim/internal/logger"
	"cryptosim/internal/model"

	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var symbolRe = regexp.MustCompile(`^[A-Z0-9]{2,6}USDT$`)

// ValidSymbol 校验交易对格式（如 BTCUSDT）。
func ValidSymbol(s string) bool { return symbolRe.MatchString(s) }

type namedUpstream struct {
	name string
	up   Upstream
}

// Service 行情服务：上游按序降级（OKX → Binance → Mock），Redis 短 TTL 缓存。
type Service struct {
	db    *gorm.DB
	rdb   *redis.Client
	mock  *Mock
	ups   []namedUpstream
	syms  []string
	mu    sync.Mutex
	fails map[string]int      // 上游连续失败次数
	until map[string]time.Time // 上游熔断截止时间
}

// NewService 创建行情服务并预加载交易对列表。
func NewService(db *gorm.DB, rdb *redis.Client, proxy string) (*Service, error) {
	s := &Service{
		db: db, rdb: rdb, mock: NewMock(),
		fails: map[string]int{}, until: map[string]time.Time{},
	}
	if okx, err := NewOKX(proxy); err == nil {
		s.ups = append(s.ups, namedUpstream{okx.Name(), okx})
	}
	if bn, err := NewBinance(proxy); err == nil {
		s.ups = append(s.ups, namedUpstream{bn.Name(), bn})
	}
	var pairs []model.TradingPair
	if err := db.Where("pair_type = ? AND enabled = ?", "spot", true).
		Order("id").Find(&pairs).Error; err == nil && len(pairs) > 0 {
		for _, p := range pairs {
			s.syms = append(s.syms, p.Symbol)
		}
	}
	if len(s.syms) == 0 {
		s.syms = []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "SOLUSDT", "XRPUSDT", "TRXUSDT", "DOGEUSDT"}
	}
	return s, nil
}

// Symbols 返回现货交易对列表。
func (s *Service) Symbols() []string {
	out := make([]string, len(s.syms))
	copy(out, s.syms)
	return out
}

// Coins 币种目录。
func (s *Service) Coins() ([]model.Coin, error) {
	var coins []model.Coin
	err := s.db.Where("enabled = ?", true).Order("sort").Find(&coins).Error
	return coins, err
}

// available 上游是否处于熔断跳过期。
func (s *Service) available(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return time.Now().After(s.until[name])
}

func (s *Service) markOK(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fails[name] = 0
}

func (s *Service) markFail(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fails[name]++
	if s.fails[name] >= 3 { // 连续失败 3 次熔断 60s，避免每次请求都等超时
		s.until[name] = time.Now().Add(60 * time.Second)
		s.fails[name] = 0
		logger.L().Warn("行情上游熔断 60s", zap.String("upstream", name))
	}
}

func (s *Service) cacheGet(ctx context.Context, key string, out any) bool {
	if s.rdb == nil {
		return false
	}
	v, err := s.rdb.Get(ctx, key).Result()
	if err != nil {
		return false
	}
	return json.Unmarshal([]byte(v), out) == nil
}

func (s *Service) cacheSet(ctx context.Context, key string, v any, ttl time.Duration) {
	if s.rdb == nil {
		return
	}
	if b, err := json.Marshal(v); err == nil {
		s.rdb.Set(ctx, key, b, ttl)
	}
}

// tryUpstreams 依次尝试可用上游（每个上游 5s 超时），全部失败返回 false。
func tryUpstreams[T any](s *Service, ctx context.Context, fn func(context.Context, Upstream) (T, error)) (T, bool) {
	var zero T
	for _, u := range s.ups {
		if !s.available(u.name) {
			continue
		}
		cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		v, err := fn(cctx, u.up)
		cancel()
		if err == nil {
			s.markOK(u.name)
			return v, true
		}
		s.markFail(u.name)
		logger.L().Debug("上游请求失败", zap.String("upstream", u.name), zap.Error(err))
	}
	return zero, false
}

// LastPrice 实时最新价（供撮合引擎等内部使用）。
func (s *Service) LastPrice(ctx context.Context, symbol string) (d decimal.Decimal, err error) {
	tks, err := s.Tickers(ctx)
	if err != nil {
		return d, err
	}
	for _, t := range tks {
		if t.Symbol == symbol {
			return decimal.NewFromString(t.Last)
		}
	}
	return d, &NotFoundError{Symbol: symbol}
}

// NotFoundError 找不到交易对行情。
type NotFoundError struct{ Symbol string }

func (e *NotFoundError) Error() string { return "no ticker for " + e.Symbol }

// Tickers 现货 24h 行情列表。
func (s *Service) Tickers(ctx context.Context) ([]Ticker, error) {
	const key = "m2:tickers"
	var cached []Ticker
	if s.cacheGet(ctx, key, &cached) && len(cached) > 0 {
		return cached, nil
	}
	if v, ok := tryUpstreams(s, ctx, func(_ context.Context, u Upstream) ([]Ticker, error) {
		return u.Tickers(ctx, s.syms)
	}); ok {
		s.cacheSet(ctx, key, v, 3*time.Second)
		return v, nil
	}
	return s.mock.Tickers(ctx, s.syms)
}

// Klines K 线数据。
func (s *Service) Klines(ctx context.Context, symbol, bar string, limit int) ([]Candle, error) {
	key := "m2:klines:" + symbol + ":" + bar + ":" + strconv.Itoa(limit)
	var cached []Candle
	if s.cacheGet(ctx, key, &cached) && len(cached) > 0 {
		return cached, nil
	}
	if v, ok := tryUpstreams(s, ctx, func(_ context.Context, u Upstream) ([]Candle, error) {
		return u.Klines(ctx, symbol, bar, limit)
	}); ok {
		s.cacheSet(ctx, key, v, 5*time.Second)
		return v, nil
	}
	return s.mock.Klines(ctx, symbol, bar, limit)
}

// Depth 盘口深度。
func (s *Service) Depth(ctx context.Context, symbol string, sz int) (*Depth, error) {
	if v, ok := tryUpstreams(s, ctx, func(_ context.Context, u Upstream) (*Depth, error) {
		return u.Depth(ctx, symbol, sz)
	}); ok {
		return v, nil
	}
	return s.mock.Depth(ctx, symbol, sz)
}

// Trades 最近成交。
func (s *Service) Trades(ctx context.Context, symbol string, limit int) ([]Trade, error) {
	if v, ok := tryUpstreams(s, ctx, func(_ context.Context, u Upstream) ([]Trade, error) {
		return u.Trades(ctx, symbol, limit)
	}); ok {
		return v, nil
	}
	return s.mock.Trades(ctx, symbol, limit)
}
