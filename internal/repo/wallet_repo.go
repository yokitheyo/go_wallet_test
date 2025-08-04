package repo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/yokitheyo/go_wallet_test/internal/config"
	"github.com/yokitheyo/go_wallet_test/internal/model"
)

type Repo struct {
	db *sql.DB
}

func NewPostgres(cfg *config.Config) (*Repo, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPass, cfg.DBName,
	)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(20)
	db.SetConnMaxLifetime(time.Minute * 5)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}
	return &Repo{db: db}, nil
}

func (r *Repo) DB() *sql.DB {
	return r.db
}

func (r *Repo) Close() error {
	return r.db.Close()
}

func (r *Repo) ChangeBalance(ctx context.Context, req model.WalletRequest) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var newBalance int64

	if req.OperationType == model.Deposit {
		err = tx.QueryRow(`
            INSERT INTO wallets(wallet_id, balance)
            VALUES ($1, $2)
            ON CONFLICT (wallet_id) DO UPDATE SET balance = wallets.balance + EXCLUDED.balance
            RETURNING balance
        `, req.WalletID, req.Amount).Scan(&newBalance)
		if err != nil {
			return 0, fmt.Errorf("failed to deposit balance: %w", err)
		}
	} else if req.OperationType == model.Withdraw {
		err = tx.QueryRow(`
            UPDATE wallets
            SET balance = balance - $1
            WHERE wallet_id = $2 AND balance >= $1
            RETURNING balance
        `, req.Amount, req.WalletID).Scan(&newBalance)

		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("insufficient balance")
		} else if err != nil {
			return 0, fmt.Errorf("failed to withdraw balance: %w", err)
		}
	} else {
		return 0, fmt.Errorf("unknown operation type")
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return newBalance, nil
}

func (r *Repo) GetBalance(walletID uuid.UUID) (int64, error) {
	var bal int64
	err := r.db.QueryRow(`SELECT balance FROM wallets WHERE wallet_id = $1`, walletID).Scan(&bal)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get balance: %w", err)
	}
	return bal, nil
}
