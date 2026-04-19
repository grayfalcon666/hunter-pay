package db

import (
	"context"
)

type AddBalanceTxParams struct {
	AccountID      int64  `json:"account_id"`
	Amount         int64  `json:"amount"`
	IdempotencyKey string `json:"idempotency_key"`
}

// AddBalanceTx is deprecated: use DepositTx instead.
// It performs a DEPOSIT: from SYSTEM account to user's account.
func (store *SQLStore) AddBalanceTx(ctx context.Context, arg AddBalanceTxParams) error {
	_, err := store.DepositTx(ctx, DepositTxParams{
		AccountID:       arg.AccountID,
		Amount:         arg.Amount,
		IdempotencyKey: arg.IdempotencyKey,
		Description:    "充值",
	})
	return err
}
