package repo

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/yokitheyo/go_wallet_test/internal/config"
	"github.com/yokitheyo/go_wallet_test/internal/model"
	"go.uber.org/zap"
)

type Repo struct {
	db     *sql.DB
	Logger *zap.Logger
}

func NewPostgres(cfg *config.Config, logger *zap.Logger) (*Repo, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPass, cfg.DBName,
	)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		logger.Error("failed to open db", zap.Error(err))
		return nil, err
	}
	if err := db.Ping(); err != nil {
		logger.Error("failed to ping db", zap.Error(err))
		return nil, err
	}
	logger.Info("database connected successfully")
	return &Repo{db: db, Logger: logger}, nil
}

func (r *Repo) DB() *sql.DB {
	return r.db
}

func (r *Repo) Close() error {
	r.Logger.Info("closing database connection")
	return r.db.Close()
}

func (r *Repo) ChangeBalance(req model.WalletRequest) (int64, error) {
	tx, err := r.db.Begin()
	if err != nil {
		r.Logger.Error("failed to begin transaction", zap.Error(err))
		return 0, err
	}
	defer tx.Rollback()

	var balance int64

	err = tx.QueryRow(`SELECT balance FROM wallets WHERE wallet_id = $1 FOR UPDATE`, req.WalletID).Scan(&balance)
	if err == sql.ErrNoRows {
		balance = 0
		_, err = tx.Exec(`INSERT INTO wallets(wallet_id, balance) VALUES($1, $2)`, req.WalletID, balance)
		if err != nil {
			r.Logger.Error("failed to insert new wallet", zap.Any("wallet_id", req.WalletID), zap.Error(err))
			return 0, err
		}
	} else if err != nil {
		r.Logger.Error("failed to select balance", zap.Any("wallet_id", req.WalletID), zap.Error(err))
		return 0, err
	}

	if req.OperationType == model.Withdraw && balance < req.Amount {
		r.Logger.Warn("insufficient balance", zap.Any("wallet_id", req.WalletID), zap.Int64("balance", balance), zap.Int64("requested", req.Amount))
		return balance, fmt.Errorf("insufficient balance")
	}

	newBal := balance
	if req.OperationType == model.Deposit {
		newBal += req.Amount
	} else {
		newBal -= req.Amount
	}

	_, err = tx.Exec(`UPDATE wallets SET balance = $1 WHERE wallet_id = $2`, newBal, req.WalletID)
	if err != nil {
		r.Logger.Error("failed to update balance", zap.Any("wallet_id", req.WalletID), zap.Int64("new_balance", newBal), zap.Error(err))
		return 0, err
	}

	if err = tx.Commit(); err != nil {
		r.Logger.Error("failed to commit transaction", zap.Error(err))
		return 0, err
	}

	r.Logger.Info("balance changed successfully", zap.Any("wallet_id", req.WalletID), zap.Int64("new_balance", newBal))
	return newBal, nil
}

func (r *Repo) GetBalance(walletID uuid.UUID) (int64, error) {
	var bal int64
	err := r.db.QueryRow(`SELECT balance FROM wallets WHERE wallet_id = $1`, walletID).Scan(&bal)
	if err == sql.ErrNoRows {
		r.Logger.Info("wallet not found, returning zero balance", zap.Any("wallet_id", walletID))
		return 0, nil
	}
	if err != nil {
		r.Logger.Error("failed to get balance", zap.Any("wallet_id", walletID), zap.Error(err))
		return 0, err
	}
	r.Logger.Info("balance retrieved", zap.Any("wallet_id", walletID), zap.Int64("balance", bal))
	return bal, nil
}
