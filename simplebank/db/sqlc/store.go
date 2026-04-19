package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type Store interface {
	Querier
	TransferTX(ctx context.Context, arg TransferTxParams) (TransferTxResult, error)
	CreateUserTx(ctx context.Context, arg CreateUserTxParams) (CreateUserTxResult, error)
	VerifyEmailTx(ctx context.Context, arg VerifyEmailTxParams) (VerifyEmailTxResult, error)
	AddBalanceTx(ctx context.Context, arg AddBalanceTxParams) error
	FreezeTx(ctx context.Context, arg FreezeTxParams) (FreezeTxResult, error)
	UnfreezeTx(ctx context.Context, arg UnfreezeTxParams) (UnfreezeTxResult, error)
	BountyPayoutTx(ctx context.Context, arg BountyPayoutTxParams) (BountyPayoutTxResult, error)
	DepositTx(ctx context.Context, arg DepositTxParams) (DepositTxResult, error)
	WithdrawFromFrozenTx(ctx context.Context, arg WithdrawFromFrozenTxParams) (WithdrawFromFrozenTxResult, error)
}

var (
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrInsufficientFrozenBalance = errors.New("insufficient frozen balance")
)

type SQLStore struct {
	*Queries
	db *sql.DB
}

func NewStore(db *sql.DB) Store {
	return &SQLStore{
		db:      db,
		Queries: New(db),
	}
}

func (store *SQLStore) execTX(ctx context.Context, fn func(*Queries) error) error {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	q := New(tx)

	err = fn(q)

	if err != nil {
		// 如果你的逻辑报错了，回滚 (Rollback)
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}
