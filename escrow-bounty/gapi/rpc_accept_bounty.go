package gapi

import (
	"context"
	"strings"

	"github.com/grayfalcon666/escrow-bounty/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (server *Server) AcceptBounty(ctx context.Context, req *pb.AcceptBountyRequest) (*pb.AcceptBountyResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}

	if req.GetBountyId() <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "非法的悬赏 ID")
	}

	// 用用户的 JWT Token 查询该用户在 SimpleBank 的账户
	accounts, err := server.bankClient.ListAccounts(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "查询账户失败: %v", err)
	}
	if len(accounts) == 0 {
		return nil, status.Errorf(codes.FailedPrecondition, "您还没有账户，请先创建账户")
	}
	hunterAccountId := accounts[0].GetId()

	// 调用包含 FOR UPDATE 行级锁的并发安全方法
	application, err := server.store.AcceptBounty(ctx, req.GetBountyId(), hunterAccountId, authPayload.Username)
	if err != nil {
		// 如果是因为违反了唯一索引（该猎人已经申请过这个悬赏）
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") ||
			strings.Contains(err.Error(), "SQLSTATE 23505") {
			return nil, status.Errorf(codes.AlreadyExists, "你已经申请过该悬赏，请勿重复操作")
		}
		return nil, status.Errorf(codes.Internal, "接单失败: %v", err)
	}

	return &pb.AcceptBountyResponse{
		Application: convertBountyApplication(application),
	}, nil
}
