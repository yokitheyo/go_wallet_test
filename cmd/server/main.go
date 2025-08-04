package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"github.com/yokitheyo/go_wallet_test/internal/config"
	"github.com/yokitheyo/go_wallet_test/internal/handler"
	"github.com/yokitheyo/go_wallet_test/internal/repo"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic("failed to initialize zap logger: " + err.Error())
	}
	defer logger.Sync()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	repository, err := repo.NewPostgres(cfg)
	if err != nil {
		logger.Fatal("db connect error", zap.Error(err))
	}
	defer func() {
		if err := repository.Close(); err != nil {
			logger.Error("failed to close database connection", zap.Error(err))
		} else {
			logger.Info("database connection closed successfully")
		}
	}()

	db := repository.DB()

	migrationsDir := filepath.Join(".", "migrations")
	if err := goose.SetDialect("postgres"); err != nil {
		logger.Fatal("goose set dialect error", zap.Error(err))
	}

	if err := goose.Up(db, migrationsDir); err != nil {
		logger.Fatal("goose migrate error", zap.Error(err))
	}
	logger.Info("database migrations completed successfully")

	// Передаем интерфейсы вместо конкретных типов
	router, gracefulShutdown := handler.NewRouter(repository, logger)

	addr := ":" + cfg.HTTPPort
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("Starting server", zap.String("address", addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("failed to run server", zap.Error(err))
		}
	}()

	sig := <-quit
	logger.Info("Received shutdown signal", zap.String("signal", sig.String()))

	logger.Info("Starting graceful shutdown...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := gracefulShutdown.Shutdown(ctx); err != nil {
		logger.Warn("Graceful shutdown middleware timeout", zap.Error(err))
	}

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	} else {
		logger.Info("Server exited gracefully")
	}

	logger.Info("Waiting for active operations to complete...")
	time.Sleep(2 * time.Second)

	logger.Info("Shutdown completed")
}
