package gapi

import (
	"context"

	db "simplebank/db/sqlc"
	"simplebank/pb"

	"github.com/lib/pq"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// 支持的货币类型（模拟 Gin 中的 custom validator）
func isSupportedCurrency(currency string) bool {
	switch currency {
	case "USD", "EUR", "CAD", "CNY": // 根据你的业务需求调整
		return true
	}
	return false
}

func (server *Server) CreateAccount(ctx context.Context, req *pb.CreateAccountRequest) (*pb.CreateAccountResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "未授权的访问: %v", err)
	}

	// 校验传入的货币类型
	if !isSupportedCurrency(req.GetCurrency()) {
		return nil, status.Errorf(codes.InvalidArgument, "不支持的货币类型: %s", req.GetCurrency())
	}

	arg := db.CreateAccountParams{
		Owner:    authPayload.Username,
		Balance:  0, // 初始余额为 0
		Currency: req.GetCurrency(),
	}

	account, err := server.store.CreateAccount(ctx, arg)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "foreign_key_violation", "unique_violation":
				return nil, status.Errorf(codes.AlreadyExists, "账户创建受限 (违反唯一或外键约束): %s", err)
			}
		}
		return nil, status.Errorf(codes.Internal, "创建账户失败: %s", err)
	}

	return &pb.CreateAccountResponse{
		Account: &pb.Account{
			Id:        account.ID,
			Owner:     account.Owner,
			Balance:   account.Balance,
			Currency:  account.Currency,
			CreatedAt: timestamppb.New(account.CreatedAt),
		},
	}, nil
}
