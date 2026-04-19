package gapi

import (
	"context"
	"database/sql"
	"errors"
	"simplebank/pb"

	db "simplebank/db/sqlc"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (server *Server) Transfer(ctx context.Context, req *pb.TransferTxRequest) (*pb.TransferTxResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, unauthenticatedError(err)
	}

	//TODO 替换validator
	if req.GetAmount() <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "转账金额必须大于 0")
	}

	fromAccount, err := server.store.GetAccount(ctx, req.GetFromAccountId())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "付款账户不存在")
		}
		return nil, status.Errorf(codes.Internal, "查询账户失败")
	}

	// 只有两种情况允许转账
	// A. 登录的用户名就是该账户的 Owner
	// B. 登录的是内部微服务特权账号 (escrow)
	if fromAccount.Owner != authPayload.Username && authPayload.Username != systemUsername {
		return nil, status.Errorf(codes.PermissionDenied, "权限不足：您无权操作他人的账户")
	}

	arg := db.TransferTxParams{
		FromAccountID:  req.GetFromAccountId(),
		ToAccountID:   req.GetToAccountId(),
		Amount:        req.GetAmount(),
		IdempotencyKey: req.GetIdempotencyKey(),
		TradeType:     req.GetTradeType(),
	}

	result, err := server.store.TransferTX(ctx, arg)
	if err != nil {
		// Idempotency key already used — this is a successful no-op (幂等成功)
		if errors.Is(err, db.ErrIdempotencyKeyAlreadyExists) {
			return &pb.TransferTxResponse{}, nil
		}
		return nil, status.Errorf(codes.Internal, "转账事务失败: %v", err)
	}

	return &pb.TransferTxResponse{
		TransferId:         result.Transfer.ID,
		FromAccountId:      result.Transfer.FromAccountID,
		ToAccountId:        result.Transfer.ToAccountID,
		Amount:             result.Transfer.Amount,
		CreatedAt:          timestamppb.New(result.Transfer.CreatedAt),
		FromAccountBalance: result.FromAccount.Balance,
		ToAccountBalance:   result.ToAccount.Balance,
	}, nil
}
