package model

import "github.com/google/uuid"

type OperationType string

const (
	Deposit  OperationType = "DEPOSIT"
	Withdraw OperationType = "WITHDRAW"
)

type WalletRequest struct {
	WalletID      uuid.UUID     `json:"walletId" binding:"required"`
	OperationType OperationType `json:"operationType" binding:"required,oneof=DEPOSIT WITHDRAW"`
	Amount        int64         `json:"amount" binding:"required,gt=0"`
}
