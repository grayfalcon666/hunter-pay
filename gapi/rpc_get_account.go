package gapi

import (
	"context"
	"database/sql"
	"errors"

	db "simplebank/db/sqlc"
	"simplebank/pb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (server *Server) GetAccount(ctx context.Context, req *pb.GetAccountRequest) (*pb.GetAccountResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, unauthenticatedError(err)
	}

	account, err := server.store.GetAccount(ctx, req.GetId())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "账户不存在")
		}
		return nil, status.Errorf(codes.Internal, "查询账户失败: %v", err)
	}

	// 能查自己的账户 (内部系统特权除外)
	if account.Owner != authPayload.Username {
		if authPayload.Username != "escrow_system" && authPayload.Username != "escrow" {
			return nil, status.Errorf(codes.PermissionDenied, "越权拦截：您无权访问他人的账户")
		}
	}

	return &pb.GetAccountResponse{
		Account: &pb.Account{
			Id:               account.ID,
			Owner:            account.Owner,
			Balance:          account.Balance,
			FrozenBalance:    account.FrozenBalance,
			AvailableBalance: account.Balance - account.FrozenBalance,
			Currency:         account.Currency,
			CreatedAt:        timestamppb.New(account.CreatedAt),
		},
	}, nil
}

func (server *Server) ListAccounts(ctx context.Context, req *pb.ListAccountsRequest) (*pb.ListAccountsResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, unauthenticatedError(err)
	}

	// 默认分页
	pageID := req.GetPageId()
	pageSize := req.GetPageSize()
	if pageID <= 0 {
		pageID = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 20 {
		pageSize = 20
	}

	accounts, err := server.store.ListAccounts(ctx, db.ListAccountsParams{
		Owner:  authPayload.Username,
		Limit:  pageSize,
		Offset: (pageID - 1) * pageSize,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "查询账户列表失败: %v", err)
	}

	pbAccounts := make([]*pb.Account, len(accounts))
	for i, acc := range accounts {
		pbAccounts[i] = &pb.Account{
			Id:               acc.ID,
			Owner:            acc.Owner,
			Balance:          acc.Balance,
			FrozenBalance:    acc.FrozenBalance,
			AvailableBalance: acc.Balance - acc.FrozenBalance,
			Currency:         acc.Currency,
			CreatedAt:        timestamppb.New(acc.CreatedAt),
		}
	}

	return &pb.ListAccountsResponse{Accounts: pbAccounts}, nil
}
