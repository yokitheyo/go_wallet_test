package middleware

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type GracefulShutdown struct {
	activeRequests int64
	shutdown       int32
	logger         *zap.Logger
}

func NewGracefulShutdown(logger *zap.Logger) *GracefulShutdown {
	return &GracefulShutdown{
		logger: logger,
	}
}

func (gs *GracefulShutdown) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if atomic.LoadInt32(&gs.shutdown) == 1 {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "server is shutting down",
			})
			c.Abort()
			return
		}

		atomic.AddInt64(&gs.activeRequests, 1)
		defer atomic.AddInt64(&gs.activeRequests, -1)

		start := time.Now()
		c.Next()
		duration := time.Since(start)

		gs.logger.Debug("Request completed",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("duration", duration),
		)
	}
}

func (gs *GracefulShutdown) Shutdown(ctx context.Context) error {
	gs.logger.Info("Initiating graceful shutdown...")

	atomic.StoreInt32(&gs.shutdown, 1)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			gs.logger.Warn("Graceful shutdown timeout reached, forcing shutdown")
			return ctx.Err()
		case <-ticker.C:
			active := atomic.LoadInt64(&gs.activeRequests)
			if active == 0 {
				gs.logger.Info("All active requests completed, shutdown successful")
				return nil
			}
			gs.logger.Info("Waiting for active requests to complete",
				zap.Int64("active_requests", active))
		}
	}
}

func (gs *GracefulShutdown) GetActiveRequests() int64 {
	return atomic.LoadInt64(&gs.activeRequests)
}

func (gs *GracefulShutdown) IsShuttingDown() bool {
	return atomic.LoadInt32(&gs.shutdown) == 1
}
