package market

import (
	"context"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Mock 离线模拟行情：几何布朗运动随机游走。
// 上游全部不可达（断网/被墙且无代理）时兜底，保证平台任何时候可用。
// 带内存序列缓存，刷新页面时历史 K 线保持一致、只增量推进。
type Mock struct {
	mu     sync.Mutex
	px     map[string]float64
	open24 map[string]float64
	day    map[string]string
	series map[string][]Candle // key: symbol|bar
	trades map[string][]Trade
	rng    *rand.Rand
}

var mockBase = map[string]float64{
	"BTCUSDT": 65000, "ETHUSDT": 3200, "BNBUSDT": 600, "SOLUSDT": 160,
	"XRPUSDT": 0.55, "TRXUSDT": 0.15, "DOGEUSDT": 0.13,
}

// NewMock 创建模拟行情源。
func NewMock() *Mock {
	return &Mock{
		px:     map[string]float64{},
		open24: map[string]float64{},
		day:    map[string]string{},
		series: map[string][]Candle{},
		trades: map[string][]Trade{},
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (m *Mock) Name() string { return "mock" }

func basePx(symbol string) float64 {
	if p, ok := mockBase[symbol]; ok {
		return p
	}
	return 100
}

// curPx 返回当前价（惰性初始化为基准价）。
func (m *Mock) curPx(symbol string) float64 {
	if p, ok := m.px[symbol]; ok {
		return p
	}
	p := basePx(symbol)
	m.px[symbol] = p
	return p
}

// f64 格式化价格：大于 1 保留 2 位，小币保留 6 位。
func f64(v float64) string {
	if v >= 1 {
		return strconv.FormatFloat(v, 'f', 2, 64)
	}
	return strconv.FormatFloat(v, 'f', 6, 64)
}

// barSeconds 周期 → 秒。
func barSeconds(bar string) int64 {
	switch bar {
	case "5m":
		return 300
	case "15m":
		return 900
	case "1h":
		return 3600
	case "4h":
		return 14400
	case "1d":
		return 86400
	default:
		return 60
	}
}

// step 单步随机游走：p *= exp(sigma * z)。
func (m *Mock) step(p, sigma float64) float64 {
	np := p * math.Exp(m.rng.NormFloat64()*sigma)
	if np <= 0 {
		return p
	}
	return np
}

// ensureKlines 取/生成 symbol 的 K 线序列（增ly推进到当前桶）。
func (m *Mock) ensureKlines(symbol, bar string, limit int) []Candle {
	key := symbol + "|" + bar
	bucket := barSeconds(bar)
	sigma := 0.004 * math.Sqrt(float64(bucket)/60)
	now := time.Now().Unix() / bucket * bucket

	cached := m.series[key]
	if len(cached) == 0 {
		p := m.curPx(symbol)
		// 从 limit-1 个桶之前正向走出来，最后落到当前价附近
		start := now - int64(limit-1)*bucket
		for ts := start; ts <= now; ts += bucket {
			p = m.step(p, sigma)
			o := p
			h := o * (1 + math.Abs(m.rng.NormFloat64())*sigma*0.6)
			l := o * (1 - math.Abs(m.rng.NormFloat64())*sigma*0.6)
			c := m.step(o, sigma*0.7)
			cached = append(cached, Candle{
				Ts: ts * 1000, O: f64(o), H: f64(math.Max(h, c)), L: f64(math.Min(l, c)), C: f64(c),
				Vol: strconv.FormatFloat(100*m.rng.Float64()+1, 'f', 3, 64),
			})
			p = c
		}
		m.px[symbol] = p
	} else {
		last := cached[len(cached)-1]
		p, _ := strconv.ParseFloat(last.C, 64)
		for ts := last.Ts/1000 + bucket; ts <= now; ts += bucket {
			p = m.step(p, sigma)
			o := p
			h := o * (1 + math.Abs(m.rng.NormFloat64())*sigma*0.6)
			l := o * (1 - math.Abs(m.rng.NormFloat64())*sigma*0.6)
			c := m.step(o, sigma*0.7)
			cached = append(cached, Candle{
				Ts: ts * 1000, O: f64(o), H: f64(math.Max(h, c)), L: f64(math.Min(l, c)), C: f64(c),
				Vol: strconv.FormatFloat(100*m.rng.Float64()+1, 'f', 3, 64),
			})
			p = c
		}
		m.px[symbol] = p
	}
	if len(cached) > limit {
		cached = cached[len(cached)-limit:]
	}
	m.series[key] = cached
	return cached
}

func (m *Mock) Tickers(_ context.Context, symbols []string) ([]Ticker, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	today := time.Now().Format("2006-01-02")
	out := make([]Ticker, 0, len(symbols))
	for _, s := range symbols {
		p := m.curPx(s)
		if m.day[s] != today { // 新交易日重置开盘价
			m.day[s] = today
			m.open24[s] = p * (1 + (m.rng.Float64()-0.5)*0.06)
		}
		o := m.open24[s]
		out = append(out, Ticker{
			Symbol: s, Last: f64(p), Open24h: f64(o),
			High24h: f64(math.Max(p, o) * 1.015), Low24h: f64(math.Min(p, o) * 0.985),
			Vol24h:    strconv.FormatFloat(10000*m.rng.Float64()+500, 'f', 3, 64),
			ChangePct: pct(f64(p), f64(o)),
		})
	}
	return out, nil
}

func (m *Mock) Klines(_ context.Context, symbol, bar string, limit int) ([]Candle, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := m.ensureKlines(symbol, bar, limit)
	cp := make([]Candle, len(out))
	copy(cp, out)
	return cp, nil
}

func (m *Mock) Depth(_ context.Context, symbol string, sz int) (*Depth, error) {
	m.mu.Lock()
	p := m.curPx(symbol)
	m.mu.Unlock()
	spread := p * 0.0004
	bids := make([]DepthLevel, 0, sz)
	asks := make([]DepthLevel, 0, sz)
	for i := 0; i < sz; i++ {
		bp := p - spread*float64(i+1)
		ap := p + spread*float64(i+1)
		bids = append(bids, DepthLevel{f64(bp), strconv.FormatFloat(0.5+m.rng.Float64()*3, 'f', 4, 64)})
		asks = append(asks, DepthLevel{f64(ap), strconv.FormatFloat(0.5+m.rng.Float64()*3, 'f', 4, 64)})
	}
	return &Depth{Bids: bids, Asks: asks}, nil
}

func (m *Mock) Trades(_ context.Context, symbol string, limit int) ([]Trade, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UnixMilli()
	list := m.trades[symbol]
	for i := 0; i < 3; i++ { // 每次请求推进 3 笔
		p := m.curPx(symbol) * (1 + (m.rng.Float64()-0.5)*0.0004)
		side := "buy"
		if m.rng.Intn(2) == 0 {
			side = "sell"
		}
		list = append(list, Trade{
			Ts: now - int64(i)*800, Price: f64(p),
			Size: strconv.FormatFloat(0.01+m.rng.Float64(), 'f', 4, 64), Side: side,
		})
	}
	if len(list) > 50 {
		list = list[len(list)-50:]
	}
	m.trades[symbol] = list
	if len(list) > limit {
		list = list[len(list)-limit:]
	}
	out := make([]Trade, len(list))
	copy(out, list)
	// 新→旧展示
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

// ensure symbol 字符串只含大写字母数字（handler 已校验，这里再兜底）。
func sanitizeSymbol(s string) string {
	return strings.TrimSpace(strings.ToUpper(s))
}
