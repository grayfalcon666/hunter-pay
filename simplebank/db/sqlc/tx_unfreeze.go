package db

import (
	"context"
	"database/sql"
	"fmt"
)

// UnfreezeTx: Unlock bounty amount (BOUNTY_REFUND) - cancel bounty
// frozen_balance decreases, balance increases
type UnfreezeTxParams struct {
	AccountID       int64
	Amount         int64
	BountyID       int64
	Description    string
	IdempotencyKey string
}

type UnfreezeTxResult struct {
	Account   Account
	Transfer Transfer
}

func (store *SQLStore) UnfreezeTx(ctx context.Context, arg UnfreezeTxParams) (UnfreezeTxResult, error) {
	var result UnfreezeTxResult

	err := store.execTX(ctx, func(q *Queries) error {
		// 1. Idempotency check — TryAcquire returns sql.ErrNoRows on duplicate
		if arg.IdempotencyKey != "" {
			if _, err := q.TryAcquireIdempotencyKey(ctx, arg.IdempotencyKey); err != nil {
				return err
			}
		}

		// 2. Check account exists and has sufficient frozen balance
		account, err := q.GetAccount(ctx, arg.AccountID)
		if err != nil {
			return fmt.Errorf("account not found: %w", err)
		}
		if account.FrozenBalance < arg.Amount {
			return ErrInsufficientFrozenBalance
		}

		// 3. Decrease frozen_balance
		updatedAccount, err := q.UnfreezeAccount(ctx, UnfreezeAccountParams{
			ID:            arg.AccountID,
			FrozenBalance: arg.Amount,
		})
		if err != nil {
			if err == sql.ErrNoRows {
				// Account existed but frozen balance changed concurrently
				return ErrInsufficientFrozenBalance
			}
			return fmt.Errorf("unfreeze account failed: %w", err)
		}

		// 3. Write BOUNTY_REFUND transfer (from=to=self, no entries written)
		transfer, err := q.CreateTransfer(ctx, CreateTransferParams{
			FromAccountID: arg.AccountID,
			ToAccountID:   arg.AccountID,
			Amount:        arg.Amount,
			TradeType:    "BOUNTY_REFUND",
			TradeID:      sql.NullInt64{Int64: arg.BountyID, Valid: true},
			Description:  sql.NullString{String: arg.Description, Valid: arg.Description != ""},
		})
		if err != nil {
			return fmt.Errorf("create refund transfer failed: %w", err)
		}

		result.Account = updatedAccount
		result.Transfer = transfer
		return nil
	})

	return result, err
}
