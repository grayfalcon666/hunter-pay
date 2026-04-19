package db

import (
	"context"
	"database/sql"
	"fmt"
)

// BountyPayoutTxParams: employer pays hunter after bounty completion
// employer: balance -= amount, frozen_balance -= amount (simultaneously)
// hunter: balance += amount
type BountyPayoutTxParams struct {
	EmployerAccountID int64
	HunterAccountID int64
	Amount         int64
	BountyID      int64
	Description   string
	IdempotencyKey string
}

type BountyPayoutTxResult struct {
	EmployerAccount Account
	HunterAccount Account
	Transfer     Transfer
	EmployerEntry Entry
	HunterEntry  Entry
}

func (store *SQLStore) BountyPayoutTx(ctx context.Context, arg BountyPayoutTxParams) (BountyPayoutTxResult, error) {
	var result BountyPayoutTxResult

	err := store.execTX(ctx, func(q *Queries) error {
		// 1. Idempotency check — TryAcquire returns sql.ErrNoRows on duplicate
		if arg.IdempotencyKey != "" {
			if _, err := q.TryAcquireIdempotencyKey(ctx, arg.IdempotencyKey); err != nil {
				return err
			}
		}

		// 2. Check employer account exists and has sufficient frozen balance
		employerAccount, err := q.GetAccount(ctx, arg.EmployerAccountID)
		if err != nil {
			return fmt.Errorf("employer account not found: %w", err)
		}
		if employerAccount.FrozenBalance < arg.Amount {
			return ErrInsufficientFrozenBalance
		}

		// 4. Deduct employer: frozen_balance -= only (balance was already deducted during Freeze)
		updatedEmployerAccount, err := q.SpendFrozenBalance(ctx, SpendFrozenBalanceParams{
			ID:            arg.EmployerAccountID,
			FrozenBalance: arg.Amount,
		})
		if err != nil {
			if err == sql.ErrNoRows {
				// Account existed but frozen balance changed concurrently
				return ErrInsufficientFrozenBalance
			}
			return fmt.Errorf("spend frozen balance failed: %w", err)
		}

		// 3. Credit hunter: balance += amount
		hunterAccount, err := q.AddAccountBalance(ctx, AddAccountBalanceParams{
			ID:     arg.HunterAccountID,
			Amount: arg.Amount,
		})
		if err != nil {
			return fmt.Errorf("credit hunter balance failed: %w", err)
		}

		// 4. Double-entry bookkeeping entries
		employerEntry, err := q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.EmployerAccountID,
			Amount:   -arg.Amount,
		})
		if err != nil {
			return fmt.Errorf("create employer entry failed: %w", err)
		}

		hunterEntry, err := q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.HunterAccountID,
			Amount:    arg.Amount,
		})
		if err != nil {
			return fmt.Errorf("create hunter entry failed: %w", err)
		}

		// 5. Transfer record: BOUNTY_PAYOUT
		transfer, err := q.CreateTransfer(ctx, CreateTransferParams{
			FromAccountID: arg.EmployerAccountID,
			ToAccountID:   arg.HunterAccountID,
			Amount:        arg.Amount,
			TradeType:    "BOUNTY_PAYOUT",
			TradeID:      sql.NullInt64{Int64: arg.BountyID, Valid: true},
			Description:  sql.NullString{String: arg.Description, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("create transfer failed: %w", err)
		}

		result.EmployerAccount = updatedEmployerAccount
		result.HunterAccount = hunterAccount
		result.Transfer = transfer
		result.EmployerEntry = employerEntry
		result.HunterEntry = hunterEntry
		return nil
	})

	return result, err
}
