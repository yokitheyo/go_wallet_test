package repo

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yokitheyo/go_wallet_test/internal/model"
)

func setupTestDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *Repo) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	repo := &Repo{
		db:     db,
		queues: make(map[uuid.UUID]chan func()),
	}

	return db, mock, repo
}

func TestRepo_GetBalance(t *testing.T) {
	db, mock, repo := setupTestDB(t)
	defer db.Close()

	walletID := uuid.New()

	t.Run("existing wallet", func(t *testing.T) {
		expectedBalance := int64(1000)
		rows := sqlmock.NewRows([]string{"balance"}).AddRow(expectedBalance)
		mock.ExpectQuery("SELECT balance FROM wallets WHERE wallet_id = \\$1").
			WithArgs(walletID).
			WillReturnRows(rows)

		balance, err := repo.GetBalance(walletID)
		assert.NoError(t, err)
		assert.Equal(t, expectedBalance, balance)
	})

	t.Run("non-existing wallet", func(t *testing.T) {
		mock.ExpectQuery("SELECT balance FROM wallets WHERE wallet_id = \\$1").
			WithArgs(walletID).
			WillReturnError(sql.ErrNoRows)

		balance, err := repo.GetBalance(walletID)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), balance)
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectQuery("SELECT balance FROM wallets WHERE wallet_id = \\$1").
			WithArgs(walletID).
			WillReturnError(sql.ErrConnDone)

		balance, err := repo.GetBalance(walletID)
		assert.Error(t, err)
		assert.Equal(t, int64(0), balance)
	})

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_ChangeBalance(t *testing.T) {
	db, mock, repo := setupTestDB(t)
	defer db.Close()

	walletID := uuid.New()
	ctx := context.Background()

	t.Run("deposit operation", func(t *testing.T) {
		req := model.WalletRequest{
			WalletID:      walletID,
			OperationType: model.Deposit,
			Amount:        100,
		}

		expectedBalance := int64(1100)
		rows := sqlmock.NewRows([]string{"balance"}).AddRow(expectedBalance)
		mock.ExpectQuery("INSERT INTO wallets").
			WithArgs(walletID, req.Amount).
			WillReturnRows(rows)

		balance, err := repo.ChangeBalance(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, expectedBalance, balance)
	})

	t.Run("withdraw operation success", func(t *testing.T) {
		req := model.WalletRequest{
			WalletID:      walletID,
			OperationType: model.Withdraw,
			Amount:        50,
		}

		expectedBalance := int64(1050)
		rows := sqlmock.NewRows([]string{"balance"}).AddRow(expectedBalance)
		mock.ExpectQuery("UPDATE wallets").
			WithArgs(req.Amount, walletID).
			WillReturnRows(rows)

		balance, err := repo.ChangeBalance(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, expectedBalance, balance)
	})

	t.Run("withdraw operation insufficient funds", func(t *testing.T) {
		req := model.WalletRequest{
			WalletID:      walletID,
			OperationType: model.Withdraw,
			Amount:        2000,
		}

		mock.ExpectQuery("UPDATE wallets").
			WithArgs(req.Amount, walletID).
			WillReturnError(sql.ErrNoRows)

		balance, err := repo.ChangeBalance(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient balance")
		assert.Equal(t, int64(0), balance)
	})

	t.Run("unknown operation type", func(t *testing.T) {
		req := model.WalletRequest{
			WalletID:      walletID,
			OperationType: "UNKNOWN",
			Amount:        100,
		}

		balance, err := repo.ChangeBalance(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown operation type")
		assert.Equal(t, int64(0), balance)
	})

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_Close(t *testing.T) {
	db, mock, repo := setupTestDB(t)
	defer db.Close()

	mock.ExpectClose()
	err := repo.Close()
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
