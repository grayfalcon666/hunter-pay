package db

import (
	"context"
	"database/sql"
	"fmt"
)

// DepositTx: Top up account from external channel (DEPOSIT)
// from = SYSTEM account (999999), to = user's account
// Writes entry (+amount) and transfer (DEPOSIT)
type DepositTxParams struct {
	AccountID       int64
	Amount         int64
	Description    string
	IdempotencyKey string
}

type DepositTxResult struct {
	Account Account
	Entry  Entry
}

const PLATFORM_ACCOUNT_ID = 999999

func (store *SQLStore) DepositTx(ctx context.Context, arg DepositTxParams) (DepositTxResult, error) {
	var result DepositTxResult

	err := store.execTX(ctx, func(q *Queries) error {
		// 1. Idempotency check — TryAcquire returns sql.ErrNoRows on duplicate
		if arg.IdempotencyKey != "" {
			if _, err := q.TryAcquireIdempotencyKey(ctx, arg.IdempotencyKey); err != nil {
				return err
			}
		}

		// 2. Add to user's balance
		account, err := q.AddAccountBalance(ctx, AddAccountBalanceParams{
			ID:     arg.AccountID,
			Amount: arg.Amount,
		})
		if err != nil {
			return fmt.Errorf("add account balance failed: %w", err)
		}

		// 3. Write entry
		entry, err := q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.AccountID,
			Amount:    arg.Amount,
		})
		if err != nil {
			return fmt.Errorf("create entry failed: %w", err)
		}

		// 4. Write DEPOSIT transfer: from=SYSTEM, to=user account
		_, err = q.CreateTransfer(ctx, CreateTransferParams{
			FromAccountID: PLATFORM_ACCOUNT_ID,
			ToAccountID:   arg.AccountID,
			Amount:        arg.Amount,
			TradeType:    "DEPOSIT",
			Description:  sql.NullString{String: arg.Description, Valid: arg.Description != ""},
		})
		if err != nil {
			return fmt.Errorf("create deposit transfer failed: %w", err)
		}

		result.Account = account
		result.Entry = entry
		return nil
	})

	return result, err
}
