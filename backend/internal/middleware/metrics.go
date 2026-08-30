// Package middleware - Prometheus 指标：HTTP 请求计数/时延 + WebSocket 连接数。
package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "cryptosim_http_requests_total", Help: "HTTP 请求总数"},
		[]string{"method", "path", "status"},
	)
	httpDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "cryptosim_http_request_duration_seconds", Help: "请求时延", Buckets: prometheus.DefBuckets},
		[]string{"method", "path"},
	)
)

func init() {
	prometheus.MustRegister(httpRequests, httpDuration)
}

// Metrics 暴露 /metrics（由 server 挂到独立路径）。
func Metrics() gin.HandlerFunc {
	return gin.WrapH(promhttp.Handler())
}

// MetricsMiddleware 请求指标采集。
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		path := c.FullPath()
		if path == "" {
			path = "unmatched"
		}
		status := strconv.Itoa(c.Writer.Status())
		httpRequests.WithLabelValues(c.Request.Method, path, status).Inc()
		httpDuration.WithLabelValues(c.Request.Method, path).Observe(time.Since(start).Seconds())
	}
}
