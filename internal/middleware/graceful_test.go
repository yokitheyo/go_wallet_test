package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func setupTestGracefulShutdown() *GracefulShutdown {
	logger, _ := zap.NewDevelopment()
	return NewGracefulShutdown(logger)
}

func TestGracefulShutdown_Middleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gs := setupTestGracefulShutdown()

	router := gin.New()
	router.Use(gs.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	t.Run("normal request", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, int64(0), gs.GetActiveRequests())
	})

	t.Run("request during shutdown", func(t *testing.T) {
		gs.shutdown = 1
		defer func() { gs.shutdown = 0 }()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})
}

func TestGracefulShutdown_Shutdown(t *testing.T) {
	gs := setupTestGracefulShutdown()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := gs.Shutdown(ctx)
	assert.NoError(t, err)
	assert.True(t, gs.IsShuttingDown())
}

func TestGracefulShutdown_ShutdownTimeout(t *testing.T) {
	gs := setupTestGracefulShutdown()

	// Симулируем активные запросы
	gs.activeRequests = 5

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := gs.Shutdown(ctx)
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
}
