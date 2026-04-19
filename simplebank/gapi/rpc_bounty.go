package gapi

import (
	"context"
	"database/sql"
	"errors"

	db "simplebank/db/sqlc"
	"simplebank/pb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// isIdempotencyDuplicate reports whether err is the idempotency key duplicate error.
func isIdempotencyDuplicate(err error) bool {
	return errors.Is(err, db.ErrIdempotencyKeyAlreadyExists) || errors.Is(err, sql.ErrNoRows)
}

func (server *Server) Freeze(ctx context.Context, req *pb.FreezeRequest) (*pb.FreezeResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, unauthenticatedError(err)
	}

	if req.GetAccountId() <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "账户ID无效")
	}
	if req.GetAmount() <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "冻结金额必须大于0")
	}

	// Verify account belongs to user
	account, err := server.store.GetAccount(ctx, req.GetAccountId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "账户不存在: %v", err)
	}
	if account.Owner != authPayload.Username && authPayload.Username != "escrow_system" && authPayload.Username != "escrow" {
		return nil, status.Errorf(codes.PermissionDenied, "无权操作该账户")
	}

	result, err := server.store.FreezeTx(ctx, db.FreezeTxParams{
		AccountID:      req.GetAccountId(),
		Amount:         req.GetAmount(),
		BountyID:       req.GetBountyId(),
		Description:    req.GetDescription(),
		IdempotencyKey: req.GetIdempotencyKey(),
	})
	if err != nil {
		if isIdempotencyDuplicate(err) {
			// Idempotency key already used — this is a successful no-op
			return &pb.FreezeResponse{
				AccountId:     req.GetAccountId(),
				FrozenBalance: account.FrozenBalance,
			}, nil
		}
		if errors.Is(err, db.ErrInsufficientBalance) {
			return nil, status.Errorf(codes.FailedPrecondition, "账户余额不足")
		}
		return nil, status.Errorf(codes.Internal, "冻结资金失败: %v", err)
	}

	return &pb.FreezeResponse{
		AccountId:     result.Account.ID,
		FrozenBalance: result.Account.FrozenBalance,
	}, nil
}

func (server *Server) Unfreeze(ctx context.Context, req *pb.UnfreezeRequest) (*pb.UnfreezeResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, unauthenticatedError(err)
	}

	if req.GetAccountId() <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "账户ID无效")
	}
	if req.GetAmount() <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "解冻金额必须大于0")
	}

	// Verify account belongs to user
	account, err := server.store.GetAccount(ctx, req.GetAccountId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "账户不存在: %v", err)
	}
	if account.Owner != authPayload.Username && authPayload.Username != "escrow_system" && authPayload.Username != "escrow" {
		return nil, status.Errorf(codes.PermissionDenied, "无权操作该账户")
	}

	result, err := server.store.UnfreezeTx(ctx, db.UnfreezeTxParams{
		AccountID:      req.GetAccountId(),
		Amount:         req.GetAmount(),
		BountyID:       req.GetBountyId(),
		Description:    req.GetDescription(),
		IdempotencyKey: req.GetIdempotencyKey(),
	})
	if err != nil {
		if isIdempotencyDuplicate(err) {
			return &pb.UnfreezeResponse{
				AccountId:     req.GetAccountId(),
				FrozenBalance: account.FrozenBalance,
			}, nil
		}
		if errors.Is(err, db.ErrInsufficientFrozenBalance) {
			return nil, status.Errorf(codes.FailedPrecondition, "冻结余额不足")
		}
		return nil, status.Errorf(codes.Internal, "解冻资金失败: %v", err)
	}

	return &pb.UnfreezeResponse{
		AccountId:     result.Account.ID,
		FrozenBalance: result.Account.FrozenBalance,
	}, nil
}

func (server *Server) BountyPayout(ctx context.Context, req *pb.BountyPayoutRequest) (*pb.BountyPayoutResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, unauthenticatedError(err)
	}

	if req.GetEmployerAccountId() <= 0 || req.GetHunterAccountId() <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "账户ID无效")
	}
	if req.GetAmount() <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "金额必须大于0")
	}

	// Verify employer account belongs to user
	employerAccount, err := server.store.GetAccount(ctx, req.GetEmployerAccountId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "雇主账户不存在: %v", err)
	}
	if employerAccount.Owner != authPayload.Username && authPayload.Username != "escrow_system" && authPayload.Username != "escrow" {
		return nil, status.Errorf(codes.PermissionDenied, "无权操作雇主账户")
	}

	result, err := server.store.BountyPayoutTx(ctx, db.BountyPayoutTxParams{
		EmployerAccountID: req.GetEmployerAccountId(),
		HunterAccountID:   req.GetHunterAccountId(),
		Amount:             req.GetAmount(),
		BountyID:          req.GetBountyId(),
		Description:       req.GetDescription(),
		IdempotencyKey:    req.GetIdempotencyKey(),
	})
	if err != nil {
		if isIdempotencyDuplicate(err) {
			return &pb.BountyPayoutResponse{
				EmployerAccountId: req.GetEmployerAccountId(),
				HunterAccountId:   req.GetHunterAccountId(),
			}, nil
		}
		return nil, status.Errorf(codes.Internal, "悬赏打款失败: %v", err)
	}

	return &pb.BountyPayoutResponse{
		EmployerAccountId: result.EmployerAccount.ID,
		HunterAccountId:   result.HunterAccount.ID,
		EmployerBalance:   result.EmployerAccount.Balance,
		HunterBalance:     result.HunterAccount.Balance,
		TransferId:        result.Transfer.ID,
	}, nil
}

func (server *Server) Deposit(ctx context.Context, req *pb.DepositRequest) (*pb.DepositResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, unauthenticatedError(err)
	}

	if req.GetAccountId() <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "账户ID无效")
	}
	if req.GetAmount() <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "充值金额必须大于0")
	}

	// Verify account belongs to user
	account, err := server.store.GetAccount(ctx, req.GetAccountId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "账户不存在: %v", err)
	}
	if account.Owner != authPayload.Username && authPayload.Username != "escrow_system" && authPayload.Username != "escrow" {
		return nil, status.Errorf(codes.PermissionDenied, "无权操作该账户")
	}

	result, err := server.store.DepositTx(ctx, db.DepositTxParams{
		AccountID:      req.GetAccountId(),
		Amount:         req.GetAmount(),
		Description:    req.GetDescription(),
		IdempotencyKey: req.GetIdempotencyKey(),
	})
	if err != nil {
		if isIdempotencyDuplicate(err) {
			return &pb.DepositResponse{
				AccountId:  req.GetAccountId(),
				NewBalance: account.Balance,
			}, nil
		}
		return nil, status.Errorf(codes.Internal, "充值失败: %v", err)
	}

	return &pb.DepositResponse{
		AccountId:  result.Account.ID,
		NewBalance: result.Account.Balance,
	}, nil
}

func (server *Server) WithdrawFromFrozen(ctx context.Context, req *pb.WithdrawFromFrozenRequest) (*pb.WithdrawFromFrozenResponse, error) {
	if req.GetAccountId() <= 0 || req.GetAmount() <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "账户ID或金额不合法")
	}

	result, err := server.store.WithdrawFromFrozenTx(ctx, db.WithdrawFromFrozenTxParams{
		AccountID:       req.GetAccountId(),
		Amount:         req.GetAmount(),
		Description:    req.GetDescription(),
		IdempotencyKey: req.GetIdempotencyKey(),
	})
	if err != nil {
		if errors.Is(err, db.ErrIdempotencyKeyAlreadyExists) {
			// Idempotency key already used — this is a successful no-op
			return &pb.WithdrawFromFrozenResponse{
				AccountId:     req.GetAccountId(),
				FrozenBalance: result.Account.FrozenBalance,
			}, nil
		}
		return nil, status.Errorf(codes.Internal, "提现扣款失败: %v", err)
	}

	return &pb.WithdrawFromFrozenResponse{
		AccountId:     result.Account.ID,
		FrozenBalance: result.Account.FrozenBalance,
	}, nil
}

// validateFreezeParams is a helper to validate freeze-related inputs
func validateFreezeParams(accountID int64, amount int64) error {
	if accountID <= 0 {
		return errors.New("account ID must be positive")
	}
	if amount <= 0 {
		return errors.New("amount must be positive")
	}
	return nil
}
