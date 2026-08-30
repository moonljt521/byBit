// CryptoSim 后端入口：加载配置 → 连库 → 迁移 → 种子 → 启动 HTTP → 优雅退出。
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cryptosim/internal/config"
	"cryptosim/internal/logger"
	"cryptosim/internal/pkg/zaperr"
	"cryptosim/internal/server"
	"cryptosim/internal/store"

	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()
	logger.Init(cfg.Env)
	defer logger.Sync()
	log := logger.L()

	st, err := store.New(context.Background(), cfg)
	if err != nil {
		log.Fatal("初始化数据库失败", zaperr.Err(err))
	}
	defer st.Close()

	if err := st.Migrate(); err != nil {
		log.Fatal("数据库迁移失败", zaperr.Err(err))
	}
	if err := st.Seed(); err != nil {
		log.Fatal("写入种子数据失败", zaperr.Err(err))
	}

	srv, err := server.New(cfg, st)
	if err != nil {
		log.Fatal("初始化行情服务失败", zaperr.Err(err))
	}
	go func() {
		log.Info("HTTP 服务启动", zap.String("addr", cfg.HTTPAddr), zap.String("env", cfg.Env))
		if err := srv.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("HTTP 服务异常退出", zaperr.Err(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("收到退出信号，优雅关闭中")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Warn("关闭超时", zaperr.Err(err))
	}
	log.Info("服务已退出")
}
