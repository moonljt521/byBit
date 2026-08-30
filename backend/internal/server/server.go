// Package server 组装路由、中间件与 HTTP 生命周期。
package server

import (
	"context"
	"net/http"
	"time"

	"cryptosim/internal/account"
	"cryptosim/internal/admin"
	"cryptosim/internal/auth"
	"cryptosim/internal/config"
	"cryptosim/internal/futures"
	"cryptosim/internal/learn"
	"cryptosim/internal/logger"
	"cryptosim/internal/market"
	"cryptosim/internal/middleware"
	"cryptosim/internal/pkg/aescrypt"
	"cryptosim/internal/pkg/jwtutil"
	"cryptosim/internal/pkg/response"
	"cryptosim/internal/pkg/zaperr"
	"cryptosim/internal/spot"
	"cryptosim/internal/store"
	"cryptosim/internal/ws"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Server struct {
	http         *http.Server
	cancelEngine context.CancelFunc
}

func New(cfg *config.Config, st *store.Store) (*Server, error) {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery(), requestLogger(), middleware.SecureHeaders(), middleware.MetricsMiddleware())
	r.Use(middleware.RateLimit(300, 10*time.Second)) // 全局限流：300 请求 / 10 秒 / IP
	r.Use(cors.New(cors.Config{
		AllowOrigins: cfg.AllowedOrigins,
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Authorization"},
		MaxAge:       12 * time.Hour,
	}))

	cryptor, err := aescrypt.New(cfg.EncKey)
	if err != nil {
		return nil, err
	}
	authLimiter := middleware.RateLimit(cfg.AuthRateLimit, time.Minute) // 登录/注册防爆破

	authSvc := auth.NewService(st.DB, cfg.InitialUSDT, cryptor)
	authH := auth.NewHandler(authSvc, cfg.JWTSecret, cfg.JWTExpire)
	accH := account.NewHandler(account.NewService(st.DB, cfg.InitialUSDT))

	marketSvc, err := market.NewService(st.DB, st.Redis, cfg.HTTPProxy)
	if err != nil {
		return nil, err
	}
	marketH := market.NewHandler(marketSvc)

	spotSvc := spot.NewService(st.DB, marketSvc.LastPrice)
	spotH := spot.NewHandler(spotSvc)

	// WebSocket 推送中心（行情广播 + 私有成交通知）
	hub := ws.NewHub(marketSvc, cfg.AllowedOrigins)
	spotSvc.Notify = hub.PublishToUser
	wsHandler := func(c *gin.Context) {
		uid := int64(0)
		if t := c.Query("token"); t != "" {
			if id, err := jwtutil.Parse(t, cfg.JWTSecret); err == nil {
				uid = id
			}
		}
		hub.Handle(uid)(c)
	}

	futSvc := futures.NewService(st.DB, marketSvc.LastPrice)
	futH := futures.NewHandler(futSvc)

	v1 := r.Group("/api/v1")
	r.GET("/metrics", middleware.Metrics())
	v1.GET("/health", func(c *gin.Context) {
		response.OK(c, gin.H{"status": "ok", "time": time.Now().Format(time.RFC3339), "version": "0.2.0"})
	})
	v1.POST("/auth/register", authLimiter, authH.Register)
	v1.POST("/auth/login", authLimiter, authH.Login)
	v1.POST("/auth/credentials/reset", middleware.Auth(cfg.JWTSecret), authH.ResetCredentials)
	v1.GET("/market/tickers", marketH.Tickers)
	v1.GET("/market/coins", marketH.Coins)
	v1.GET("/market/klines", marketH.Klines)
	v1.GET("/market/depth", marketH.Depth)
	v1.GET("/market/trades", marketH.Trades)

	acct := v1.Group("/account", middleware.Private(cfg.JWTSecret, st.DB, cryptor))
	acct.GET("/me", accH.Me)
	acct.POST("/reset", accH.Reset)

	spotG := v1.Group("/spot", middleware.Private(cfg.JWTSecret, st.DB, cryptor))
	spotG.POST("/orders", spotH.Place)
	spotG.DELETE("/orders/:id", spotH.Cancel)
	spotG.GET("/orders/open", spotH.OpenOrders)
	spotG.GET("/orders/history", spotH.History)
	spotG.GET("/trades", spotH.MyTrades)

	futG := v1.Group("/futures", middleware.Private(cfg.JWTSecret, st.DB, cryptor))
	futG.POST("/positions", futH.Open)
	futG.POST("/positions/:id/close", futH.Close)
	futG.GET("/positions", futH.Positions)
	futG.GET("/positions/history", futH.History)
	futG.GET("/funding", futH.Funding)

	learnSvc := learn.NewService(cfg.LearnDir)
	learnH := learn.NewHandler(learnSvc)
	v1.GET("/ws/market", wsHandler)
	v1.GET("/learn/coins", learnH.Coins)
	v1.GET("/learn/coins/:slug", learnH.Coin)
	v1.GET("/learn/concepts", learnH.Concepts)
	v1.GET("/learn/concepts/:slug", learnH.Concept)
	v1.GET("/learn/glossary", learnH.Glossary)

	adminH := admin.NewHandler(admin.NewService(st.DB))
	adm := v1.Group("/admin", middleware.PrivateAdmin(cfg.JWTSecret, st.DB, cryptor))
	adm.GET("/stats", adminH.Stats)
	adm.GET("/users", adminH.Users)
	adm.PATCH("/users/:id/status", adminH.SetStatus)
	adm.POST("/users/:id/grant", adminH.AdjustFunds)
	adm.GET("/ledger", adminH.Ledger)
	adm.GET("/login-logs", adminH.LoginLogs)

	engCtx, cancelEngine := context.WithCancel(context.Background())
	go spot.NewEngine(spotSvc).Run(engCtx)
	go futures.NewEngine(futSvc).Run(engCtx)
	go hub.Run(engCtx)

	return &Server{
		http: &http.Server{
			Addr:              cfg.HTTPAddr,
			Handler:           r,
			ReadHeaderTimeout: 5 * time.Second,
		},
		cancelEngine: cancelEngine,
	}, nil
}

func (s *Server) Run() error { return s.http.ListenAndServe() }

// Shutdown 先停撮合引擎再停 HTTP，避免关闭中还在写库。
func (s *Server) Shutdown(ctx context.Context) error {
	if s.cancelEngine != nil {
		s.cancelEngine()
	}
	return s.http.Shutdown(ctx)
}

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		logger.L().Info("http",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
			zap.String("ip", c.ClientIP()),
			zaperr.Err(c.Errors.Last()),
		)
	}
}
