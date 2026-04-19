package db

import (
	"context"
	"database/sql"
	"errors"
)

// ErrIdempotencyKeyAlreadyExists is returned when an idempotency key has already been used,
// meaning the operation was already applied.
var ErrIdempotencyKeyAlreadyExists = errors.New("idempotency key already exists")

type TransferTxParams struct {
	FromAccountID  int64 `json:"from_account_id"`
	ToAccountID   int64 `json:"to_account_id"`
	Amount        int64 `json:"amount"`
	IdempotencyKey string `json:"idempotency_key"`
	TradeType      string `json:"trade_type"`
}

// 转账事务输出结果结构体
type TransferTxResult struct {
	Transfer    Transfer `json:"transfer"`
	FromAccount Account  `json:"from_account"`
	ToAccount   Account  `json:"to_account"`
	FromEntry   Entry    `json:"from_entry"`
	ToEntry     Entry    `json:"to_entry"`
}

// TransferTX performs a double-entry transfer between two accounts.
// If arg.IdempotencyKey is non-empty, duplicate calls with the same key will
// return sql.ErrNoRows (ErrDuplicateIdempotencyKey) without modifying any data.
func (store *SQLStore) TransferTX(ctx context.Context, arg TransferTxParams) (TransferTxResult, error) {
	var result TransferTxResult

	err := store.execTX(ctx, func(q *Queries) error {
		var err error

		// 1. Idempotency check — must be first to prevent double-entry on retry
		if arg.IdempotencyKey != "" {
			_, err = q.TryAcquireIdempotencyKey(ctx, arg.IdempotencyKey)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return ErrIdempotencyKeyAlreadyExists
				}
				return err
			}
		}

		tradeType := arg.TradeType
		if tradeType == "" {
			tradeType = "TRANSFER"
		}

		result.Transfer, err = q.CreateTransfer(ctx, CreateTransferParams{
			FromAccountID: arg.FromAccountID,
			ToAccountID:   arg.ToAccountID,
			Amount:        arg.Amount,
			TradeType:     tradeType,
		})
		if err != nil {
			return err
		}

		result.FromEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.FromAccountID,
			Amount:    -arg.Amount,
		})
		if err != nil {
			return err
		}

		result.ToEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.ToAccountID,
			Amount:    arg.Amount,
		})
		if err != nil {
			return err
		}

		if arg.FromAccountID < arg.ToAccountID {
			result.FromAccount, result.ToAccount, err = addMoney(
				ctx, q,
				arg.FromAccountID, -arg.Amount,
				arg.ToAccountID, arg.Amount,
			)
		} else {
			result.ToAccount, result.FromAccount, err = addMoney(
				ctx, q,
				arg.ToAccountID, arg.Amount,
				arg.FromAccountID, -arg.Amount,
			)
		}

		return err
	})

	return result, err
}

func addMoney(
	ctx context.Context,
	q *Queries,
	accountID1 int64,
	amount1 int64,
	accountID2 int64,
	amount2 int64,
) (account1 Account, account2 Account, err error) {
	account1, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
		ID:     accountID1,
		Amount: amount1,
	})
	if err != nil {
		return
	}

	account2, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
		ID:     accountID2,
		Amount: amount2,
	})
	return
}
