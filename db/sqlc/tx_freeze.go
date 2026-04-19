package db

import (
	"context"
	"database/sql"
	"fmt"
)

// FreezeTx: Lock bounty amount in employer's account (BOUNTY_FREEZE)
// balance decreases, frozen_balance increases
type FreezeTxParams struct {
	AccountID       int64
	Amount         int64
	BountyID       int64
	Description    string
	IdempotencyKey string
}

type FreezeTxResult struct {
	Account       Account
	Transfer     Transfer
}

func (store *SQLStore) FreezeTx(ctx context.Context, arg FreezeTxParams) (FreezeTxResult, error) {
	var result FreezeTxResult

	err := store.execTX(ctx, func(q *Queries) error {
		// 1. Idempotency check — TryAcquire returns sql.ErrNoRows on duplicate
		if arg.IdempotencyKey != "" {
			if _, err := q.TryAcquireIdempotencyKey(ctx, arg.IdempotencyKey); err != nil {
				return err
			}
		}

		// 2. Check account exists and has sufficient balance
		account, err := q.GetAccount(ctx, arg.AccountID)
		if err != nil {
			return fmt.Errorf("account not found: %w", err)
		}
		if account.Balance < arg.Amount {
			return ErrInsufficientBalance
		}

		// 3. Decrease balance and increase frozen_balance
		updatedAccount, err := q.FreezeAccount(ctx, FreezeAccountParams{
			ID:      arg.AccountID,
			Balance: arg.Amount,
		})
		if err != nil {
			if err == sql.ErrNoRows {
				// Account existed but balance changed concurrently
				return ErrInsufficientBalance
			}
			return fmt.Errorf("freeze account failed: %w", err)
		}

		// 3. Write BOUNTY_FREEZE transfer (from=to=self, no entries written)
		transfer, err := q.CreateTransfer(ctx, CreateTransferParams{
			FromAccountID: arg.AccountID,
			ToAccountID:   arg.AccountID,
			Amount:        arg.Amount,
			TradeType:    "BOUNTY_FREEZE",
			TradeID:      sql.NullInt64{Int64: arg.BountyID, Valid: true},
			Description:  sql.NullString{String: arg.Description, Valid: arg.Description != ""},
		})
		if err != nil {
			return fmt.Errorf("create freeze transfer failed: %w", err)
		}

		result.Account = updatedAccount
		result.Transfer = transfer
		return nil
	})

	return result, err
}
