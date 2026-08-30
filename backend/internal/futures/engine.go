package futures

import (
	"context"
	"time"

	"cryptosim/internal/logger"

	"go.uber.org/zap"
)

// Engine 合约引擎：每秒检查强平；每小时批量结算到期的资金费率（费率周期 8h）。
type Engine struct{ svc *Service }

func NewEngine(svc *Service) *Engine { return &Engine{svc: svc} }

func (e *Engine) Run(ctx context.Context) {
	logger.L().Info("合约引擎启动")
	t := time.NewTicker(time.Second)
	funding := time.NewTicker(time.Hour)
	defer t.Stop()
	defer funding.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.L().Info("合约引擎停止")
			return
		case <-t.C:
			if n, err := e.svc.Liquidate(ctx); err != nil {
				logger.L().Warn("强平检查失败", zap.Error(err))
			} else if n > 0 {
				logger.L().Info("触发强平", zap.Int("count", n))
			}
		case <-funding.C:
			if n, err := e.svc.SettleFunding(ctx); err != nil {
				logger.L().Warn("资金费率结算失败", zap.Error(err))
			} else if n > 0 {
				logger.L().Info("资金费率结算", zap.Int("count", n))
			}
		}
	}
}
