package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// WithdrawFromFrozenTxParams: permanently deduct from frozen_balance for withdrawal
// This is called when Alipay confirms the withdrawal was successful.
// Writes a WITHDRAWAL_OUT transfer record.
type WithdrawFromFrozenTxParams struct {
	AccountID       int64
	Amount         int64
	Description    string
	IdempotencyKey string
}

type WithdrawFromFrozenTxResult struct {
	Account Account
}

func (store *SQLStore) WithdrawFromFrozenTx(ctx context.Context, arg WithdrawFromFrozenTxParams) (WithdrawFromFrozenTxResult, error) {
	var result WithdrawFromFrozenTxResult

	err := store.execTX(ctx, func(q *Queries) error {
		// 1. Idempotency check — TryAcquire returns sql.ErrNoRows on duplicate
		if arg.IdempotencyKey != "" {
			if _, err := q.TryAcquireIdempotencyKey(ctx, arg.IdempotencyKey); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return ErrIdempotencyKeyAlreadyExists
				}
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

		// 3. Decrease frozen_balance (permanent deduction)
		updatedAccount, err := q.SpendFrozenForWithdrawal(ctx, SpendFrozenForWithdrawalParams{
			ID:            arg.AccountID,
			FrozenBalance: arg.Amount,
		})
		if err != nil {
			if err == sql.ErrNoRows {
				// Account existed but frozen balance changed concurrently
				return ErrInsufficientFrozenBalance
			}
			return fmt.Errorf("spend frozen for withdrawal failed: %w", err)
		}

		// 3. Write WITHDRAWAL_OUT transfer record (no entries — no real money moved)
		_, err = q.CreateTransfer(ctx, CreateTransferParams{
			FromAccountID: arg.AccountID,
			ToAccountID:   arg.AccountID,
			Amount:        arg.Amount,
			TradeType:     "WITHDRAWAL_OUT",
			Description:   sql.NullString{String: arg.Description, Valid: arg.Description != ""},
		})
		if err != nil {
			return fmt.Errorf("create withdrawal transfer failed: %w", err)
		}

		result.Account = updatedAccount
		return nil
	})

	return result, err
}
