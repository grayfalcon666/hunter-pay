package gapi

import (
	"context"

	"github.com/grayfalcon666/escrow-bounty/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (server *Server) CompleteBounty(ctx context.Context, req *pb.CompleteBountyRequest) (*pb.CompleteBountyResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}

	err = server.store.CompleteBounty(ctx, req.GetBountyId(), authPayload.Username, server.bankClient)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "完成悬赏失败: %v", err)
	}

	// 查询悬赏详情（含猎人信息）
	bounty, err := server.store.GetBountyByID(ctx, req.GetBountyId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "查询悬赏详情失败: %v", err)
	}

	return &pb.CompleteBountyResponse{Bounty: convertBounty(bounty)}, nil
}
