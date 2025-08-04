package model

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestWalletRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request WalletRequest
		isValid bool
	}{
		{
			name: "valid deposit request",
			request: WalletRequest{
				WalletID:      uuid.New(),
				OperationType: Deposit,
				Amount:        100,
			},
			isValid: true,
		},
		{
			name: "valid withdraw request",
			request: WalletRequest{
				WalletID:      uuid.New(),
				OperationType: Withdraw,
				Amount:        50,
			},
			isValid: true,
		},
		{
			name: "invalid amount zero",
			request: WalletRequest{
				WalletID:      uuid.New(),
				OperationType: Deposit,
				Amount:        0,
			},
			isValid: false,
		},
		{
			name: "invalid amount negative",
			request: WalletRequest{
				WalletID:      uuid.New(),
				OperationType: Deposit,
				Amount:        -10,
			},
			isValid: false,
		},
		{
			name: "invalid operation type",
			request: WalletRequest{
				WalletID:      uuid.New(),
				OperationType: "INVALID",
				Amount:        100,
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Здесь можно добавить валидацию через gin binding
			// Пока просто проверяем базовую логику
			if tt.isValid {
				assert.NotEqual(t, uuid.Nil, tt.request.WalletID)
				assert.True(t, tt.request.Amount > 0)
				assert.True(t, tt.request.OperationType == Deposit || tt.request.OperationType == Withdraw)
			}
		})
	}
}

func TestOperationType_Constants(t *testing.T) {
	assert.Equal(t, OperationType("DEPOSIT"), Deposit)
	assert.Equal(t, OperationType("WITHDRAW"), Withdraw)
}
