package repo

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/yokitheyo/go_wallet_test/internal/config"
	"github.com/yokitheyo/go_wallet_test/internal/model"
)

type Repo struct {
	db     *sql.DB
	mu     sync.Mutex
	queues map[uuid.UUID]chan func()
	wg     sync.WaitGroup
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

	db.SetMaxOpenConns(200)
	db.SetMaxIdleConns(50)
	db.SetConnMaxLifetime(time.Minute * 5)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}

	return &Repo{
		db:     db,
		queues: make(map[uuid.UUID]chan func()),
	}, nil
}

func (r *Repo) DB() *sql.DB {
	return r.db
}

func (r *Repo) Close() error {
	r.wg.Wait()
	return r.db.Close()
}

func (r *Repo) getQueue(walletID uuid.UUID) chan func() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if ch, ok := r.queues[walletID]; ok {
		return ch
	}

	ch := make(chan func(), 1000)
	r.queues[walletID] = ch

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		for job := range ch {
			job()
		}
	}()

	return ch
}

func (r *Repo) ChangeBalance(ctx context.Context, req model.WalletRequest) (int64, error) {
	resultChan := make(chan struct {
		balance int64
		err     error
	}, 1)

	q := r.getQueue(req.WalletID)

	q <- func() {
		newBalance, err := r.changeBalanceAtomic(ctx, req)
		resultChan <- struct {
			balance int64
			err     error
		}{newBalance, err}
	}

	res := <-resultChan
	return res.balance, res.err
}

func (r *Repo) changeBalanceAtomic(ctx context.Context, req model.WalletRequest) (int64, error) {
	var newBalance int64
	var err error

	switch req.OperationType {
	case model.Deposit:
		err = r.db.QueryRowContext(ctx, `
			INSERT INTO wallets(wallet_id, balance)
			VALUES ($1, $2)
			ON CONFLICT (wallet_id) DO UPDATE
			SET balance = wallets.balance + EXCLUDED.balance
			RETURNING balance
		`, req.WalletID, req.Amount).Scan(&newBalance)

	case model.Withdraw:
		err = r.db.QueryRowContext(ctx, `
			UPDATE wallets
			SET balance = balance - $1
			WHERE wallet_id = $2 AND balance >= $1
			RETURNING balance
		`, req.Amount, req.WalletID).Scan(&newBalance)
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("insufficient balance")
		}

	default:
		return 0, fmt.Errorf("unknown operation type")
	}

	return newBalance, err
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
