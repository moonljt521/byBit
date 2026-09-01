// Package ws WebSocket 推送中心：
//
//	GET /ws/market?channels=tickers[,kline:BTCUSDT]&token=<JWT>
//	channels:
//	  tickers          公开：全交易对 24h 行情快照（约每 3 秒）
//	  private:{uid}    私有：本人订单成交通知（需 token 且 uid 匹配）
package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"cryptosim/internal/logger"
	"cryptosim/internal/market"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	channels map[string]bool
	uid      int64
}

type Hub struct {
	mu       sync.RWMutex
	clients  map[*Client]bool
	upgrader websocket.Upgrader
	origins  []string
	market   *market.Service
}

func NewHub(marketSvc *market.Service, origins []string) *Hub {
	return &Hub{
		clients: map[*Client]bool{},
		market:  marketSvc,
		origins: origins,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true // 非浏览器客户端
				}
				for _, o := range origins {
					if o == origin {
						return true
					}
				}
				return false
			},
		},
	}
}

func (h *Hub) add(c *Client) {
	h.mu.Lock()
	h.clients[c] = true
	h.mu.Unlock()
}

func (h *Hub) remove(c *Client) {
	h.mu.Lock()
	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
		close(c.send)
	}
	h.mu.Unlock()
}

// Broadcast 向订阅了 channel 的客户端推送。
func (h *Hub) Broadcast(channel string, payload any) {
	data, err := json.Marshal(map[string]any{"channel": channel, "data": payload, "ts": time.Now().UnixMilli()})
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		if !c.channels[channel] && !c.channels["*"] {
			continue
		}
		select {
		case c.send <- data:
		default: // 发送缓冲满则丢弃（行情推送可容忍）
		}
	}
}

// PublishToUser 私有事件推送（成交通知等）。
func (h *Hub) PublishToUser(uid int64, payload any) {
	h.Broadcast(privateChannel(uid), payload)
}

func privateChannel(uid int64) string { return "private" }

// Run 广播循环：行情快照 + 心跳清理。
func (h *Hub) Run(ctx context.Context) {
	logger.L().Info("WebSocket Hub 启动")
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.L().Info("WebSocket Hub 停止")
			return
		case <-ticker.C:
			if tks, err := h.market.Tickers(ctx); err == nil {
				h.Broadcast("tickers", tks)
			}
		}
	}
}

// Handle godoc: GET /ws/market
func (h *Hub) Handle(jwtUID int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !h.upgrader.CheckOrigin(c.Request) {
			c.JSON(http.StatusForbidden, gin.H{"code": 10403, "msg": "Origin 不被允许"})
			return
		}
		conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			logger.L().Warn("WS 升级失败", zap.Error(err))
			return
		}
		channels := map[string]bool{}
		for _, ch := range strings.Split(c.Query("channels"), ",") {
			ch = strings.TrimSpace(ch)
			if ch == "" {
				continue
			}
			// private 频道必须携带合法 token 且 uid 一致
			if ch == "private" {
				if jwtUID == 0 {
					continue
				}
				ch = privateChannel(jwtUID)
			}
			channels[ch] = true
		}
		client := &Client{
			hub: h, conn: conn, channels: channels, uid: jwtUID,
			send: make(chan []byte, 64),
		}
		h.add(client)
		go h.writePump(client)
		go h.readPump(client)
	}
}

func (h *Hub) readPump(c *Client) {
	defer func() {
		h.remove(c)
		_ = c.conn.Close()
	}()
	c.conn.SetReadLimit(4096)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (h *Hub) writePump(c *Client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		h.remove(c)
		_ = c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
