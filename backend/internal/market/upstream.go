// Package market 行情模块：OKX / Binance 公开行情适配（按序降级）+ 离线模拟兜底。
// 上游仅使用无需密钥的公开只读接口。
package market

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Ticker 24h 行情快照（价格用字符串保留精度，前端展示时转换）。
type Ticker struct {
	Symbol    string `json:"symbol"`
	Last      string `json:"last"`
	Open24h   string `json:"open24h"`
	High24h   string `json:"high24h"`
	Low24h    string `json:"low24h"`
	Vol24h    string `json:"vol24h"` // 成交量（base 币）
	ChangePct string `json:"change_pct"`
}

// Candle K 线，按时间升序。
type Candle struct {
	Ts  int64  `json:"ts"`
	O   string `json:"o"`
	H   string `json:"h"`
	L   string `json:"l"`
	C   string `json:"c"`
	Vol string `json:"vol"`
}

// DepthLevel 盘口档位 [价格, 数量]。
type DepthLevel []string

type Depth struct {
	Bids []DepthLevel `json:"bids"`
	Asks []DepthLevel `json:"asks"`
}

type Trade struct {
	Ts    int64  `json:"ts"`
	Price string `json:"price"`
	Size  string `json:"size"`
	Side  string `json:"side"` // buy / sell（taker 方向）
}

// Bars 对外统一的 K 线周期（小写）。
var Bars = map[string]bool{"1m": true, "5m": true, "15m": true, "1h": true, "4h": true, "1d": true}

// Upstream 行情上游适配器。
type Upstream interface {
	Name() string
	Tickers(ctx context.Context, symbols []string) ([]Ticker, error)
	Klines(ctx context.Context, symbol, bar string, limit int) ([]Candle, error)
	Depth(ctx context.Context, symbol string, sz int) (*Depth, error)
	Trades(ctx context.Context, symbol string, limit int) ([]Trade, error)
}

func newHTTPClient(proxy string) (*http.Client, error) {
	t := &http.Transport{
		MaxIdleConns:        20,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
	}
	if proxy != "" {
		u, err := url.Parse(proxy)
		if err != nil {
			return nil, fmt.Errorf("代理地址无效: %w", err)
		}
		t.Proxy = http.ProxyURL(u)
	} else {
		t.Proxy = http.ProxyFromEnvironment
	}
	return &http.Client{Timeout: 6 * time.Second, Transport: t}, nil
}

// pct 计算涨跌幅字符串。(last-open)/open*100
func pct(last, open string) string {
	l, err1 := strconv.ParseFloat(last, 64)
	o, err2 := strconv.ParseFloat(open, 64)
	if err1 != nil || err2 != nil || o == 0 {
		return "0"
	}
	return strconv.FormatFloat((l-o)/o*100, 'f', 2, 64)
}

// ---------------- OKX ----------------

type okxClient struct {
	base string
	http *http.Client
}

// NewOKX 创建 OKX 公开行情适配器。
func NewOKX(proxy string) (Upstream, error) {
	c, err := newHTTPClient(proxy)
	if err != nil {
		return nil, err
	}
	return &okxClient{base: "https://www.okx.com", http: c}, nil
}

func (c *okxClient) Name() string { return "okx" }

// okxInstId BTCUSDT → BTC-USDT（本项目交易对报价币固定为 USDT）。
func okxInstId(symbol string) string {
	return strings.TrimSuffix(symbol, "USDT") + "-USDT"
}

// okxBar 统一小写周期 → OKX 大写周期。
var okxBar = map[string]string{"1m": "1m", "5m": "5m", "15m": "15m", "1h": "1H", "4h": "4H", "1d": "1D"}

type okxResp struct {
	Code string          `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func (c *okxClient) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("okx http %d", resp.StatusCode)
	}
	var r okxResp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return err
	}
	if r.Code != "0" {
		return fmt.Errorf("okx code=%s msg=%s", r.Code, r.Msg)
	}
	return json.Unmarshal(r.Data, out)
}

type okxTicker struct {
	InstId  string `json:"instId"`
	Last    string `json:"last"`
	Open24h string `json:"open24h"`
	High24h string `json:"high24h"`
	Low24h  string `json:"low24h"`
	Vol24h  string `json:"vol24h"`
}

func (c *okxClient) Tickers(ctx context.Context, symbols []string) ([]Ticker, error) {
	want := make(map[string]bool, len(symbols))
	for _, s := range symbols {
		want[s] = true
	}
	var rows []okxTicker
	if err := c.get(ctx, "/api/v5/market/tickers?instType=SPOT", &rows); err != nil {
		return nil, err
	}
	out := make([]Ticker, 0, len(symbols))
	for _, r := range rows {
		sym := strings.ReplaceAll(r.InstId, "-", "")
		if !want[sym] {
			continue
		}
		out = append(out, Ticker{
			Symbol: sym, Last: r.Last, Open24h: r.Open24h,
			High24h: r.High24h, Low24h: r.Low24h, Vol24h: r.Vol24h,
			ChangePct: pct(r.Last, r.Open24h),
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("okx 未返回所需交易对")
	}
	return out, nil
}

func (c *okxClient) Klines(ctx context.Context, symbol, bar string, limit int) ([]Candle, error) {
	b, ok := okxBar[bar]
	if !ok {
		return nil, fmt.Errorf("不支持的周期 %s", bar)
	}
	var rows [][]string
	path := fmt.Sprintf("/api/v5/market/candles?instId=%s&bar=%s&limit=%d", okxInstId(symbol), b, limit)
	if err := c.get(ctx, path, &rows); err != nil {
		return nil, err
	}
	// OKX 返回新→旧，翻转并裁掉未确认桶
	out := make([]Candle, 0, len(rows))
	for i := len(rows) - 1; i >= 0; i-- {
		r := rows[i]
		if len(r) < 6 {
			continue
		}
		ts, _ := strconv.ParseInt(r[0], 10, 64)
		out = append(out, Candle{Ts: ts, O: r[1], H: r[2], L: r[3], C: r[4], Vol: r[5]})
	}
	return out, nil
}

func (c *okxClient) Depth(ctx context.Context, symbol string, sz int) (*Depth, error) {
	var books []struct {
		Bids [][]string `json:"bids"`
		Asks [][]string `json:"asks"`
	}
	path := fmt.Sprintf("/api/v5/market/books?instId=%s&sz=%d", okxInstId(symbol), sz)
	if err := c.get(ctx, path, &books); err != nil {
		return nil, err
	}
	if len(books) == 0 {
		return nil, fmt.Errorf("okx 盘口为空")
	}
	return &Depth{Bids: toLevels(books[0].Bids), Asks: toLevels(books[0].Asks)}, nil
}

func (c *okxClient) Trades(ctx context.Context, symbol string, limit int) ([]Trade, error) {
	var rows []struct {
		Px   string `json:"px"`
		Sz   string `json:"sz"`
		Side string `json:"side"`
		Ts   string `json:"ts"`
	}
	path := fmt.Sprintf("/api/v5/market/trades?instId=%s&limit=%d", okxInstId(symbol), limit)
	if err := c.get(ctx, path, &rows); err != nil {
		return nil, err
	}
	out := make([]Trade, 0, len(rows))
	for _, r := range rows {
		ts, _ := strconv.ParseInt(r.Ts, 10, 64)
		out = append(out, Trade{Ts: ts, Price: r.Px, Size: r.Sz, Side: r.Side})
	}
	return out, nil
}

func toLevels(raw [][]string) []DepthLevel {
	out := make([]DepthLevel, 0, len(raw))
	for _, r := range raw {
		if len(r) >= 2 {
			out = append(out, DepthLevel{r[0], r[1]})
		}
	}
	return out
}

// ---------------- Binance ----------------

type binanceClient struct {
	base string
	http *http.Client
}

// NewBinance 创建 Binance 公开行情适配器（备用上游）。
func NewBinance(proxy string) (Upstream, error) {
	c, err := newHTTPClient(proxy)
	if err != nil {
		return nil, err
	}
	return &binanceClient{base: "https://api.binance.com", http: c}, nil
}

func (c *binanceClient) Name() string { return "binance" }

func (c *binanceClient) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("binance http %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

type binanceTicker struct {
	Symbol             string `json:"symbol"`
	LastPrice          string `json:"lastPrice"`
	OpenPrice          string `json:"openPrice"`
	HighPrice          string `json:"highPrice"`
	LowPrice           string `json:"lowPrice"`
	Volume             string `json:"volume"`
	PriceChangePercent string `json:"priceChangePercent"`
}

func (c *binanceClient) Tickers(ctx context.Context, symbols []string) ([]Ticker, error) {
	raw, _ := json.Marshal(symbols)
	var rows []binanceTicker
	if err := c.get(ctx, "/api/v3/ticker/24hr?symbols="+url.QueryEscape(string(raw)), &rows); err != nil {
		return nil, err
	}
	out := make([]Ticker, 0, len(rows))
	for _, r := range rows {
		out = append(out, Ticker{
			Symbol: r.Symbol, Last: r.LastPrice, Open24h: r.OpenPrice,
			High24h: r.HighPrice, Low24h: r.LowPrice, Vol24h: r.Volume,
			ChangePct: strings.TrimSuffix(r.PriceChangePercent, "00"), // "1.23450000" 已是百分数，保留 2 位
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("binance 未返回交易对")
	}
	return out, nil
}

func (c *binanceClient) Klines(ctx context.Context, symbol, bar string, limit int) ([]Candle, error) {
	var rows [][]any
	path := fmt.Sprintf("/api/v3/klines?symbol=%s&interval=%s&limit=%d", symbol, bar, limit)
	if err := c.get(ctx, path, &rows); err != nil {
		return nil, err
	}
	out := make([]Candle, 0, len(rows))
	for _, r := range rows {
		if len(r) < 6 {
			continue
		}
		ts, _ := r[0].(float64)
		out = append(out, Candle{
			Ts:  int64(ts),
			O:   asStr(r[1]),
			H:   asStr(r[2]),
			L:   asStr(r[3]),
			C:   asStr(r[4]),
			Vol: asStr(r[5]),
		})
	}
	return out, nil
}

func asStr(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return "0"
	}
}

func (c *binanceClient) Depth(ctx context.Context, symbol string, sz int) (*Depth, error) {
	var d struct {
		Bids [][]string `json:"bids"`
		Asks [][]string `json:"asks"`
	}
	path := fmt.Sprintf("/api/v3/depth?symbol=%s&limit=%d", symbol, sz)
	if err := c.get(ctx, path, &d); err != nil {
		return nil, err
	}
	return &Depth{Bids: toLevels(d.Bids), Asks: toLevels(d.Asks)}, nil
}

func (c *binanceClient) Trades(ctx context.Context, symbol string, limit int) ([]Trade, error) {
	var rows []struct {
		Price        string `json:"price"`
		Qty          string `json:"qty"`
		Time         int64  `json:"time"`
		IsBuyerMaker bool   `json:"isBuyerMaker"`
	}
	path := fmt.Sprintf("/api/v3/trades?symbol=%s&limit=%d", symbol, limit)
	if err := c.get(ctx, path, &rows); err != nil {
		return nil, err
	}
	out := make([]Trade, 0, len(rows))
	for _, r := range rows {
		side := "buy"
		if r.IsBuyerMaker {
			side = "sell" // 买方为挂单方 → 主动成交是卖出
		}
		out = append(out, Trade{Ts: r.Time, Price: r.Price, Size: r.Qty, Side: side})
	}
	return out, nil
}
