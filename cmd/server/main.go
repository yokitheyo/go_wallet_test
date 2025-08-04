package main

import (
	"path/filepath"

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

	repo, err := repo.NewPostgres(cfg)
	if err != nil {
		logger.Fatal("db connect error", zap.Error(err))
	}
	defer repo.Close()

	db := repo.DB()

	migrationsDir := filepath.Join(".", "migrations")
	if err := goose.SetDialect("postgres"); err != nil {
		logger.Fatal("goose set dialect error", zap.Error(err))
	}

	if err := goose.Up(db, migrationsDir); err != nil {
		logger.Fatal("goose migrate error", zap.Error(err))
	}

	router := handler.NewRouter(repo, logger)
	addr := ":" + cfg.HTTPPort
	logger.Info("Starting server", zap.String("address", addr))

	if err := router.Run(addr); err != nil {
		logger.Fatal("failed to run server", zap.Error(err))
	}
}
