package gapi

import (
	"context"

	"github.com/grayfalcon666/escrow-bounty/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (server *Server) ConfirmHunter(ctx context.Context, req *pb.ConfirmHunterRequest) (*pb.ConfirmHunterResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}

	err = server.store.ConfirmHunter(ctx, req.GetBountyId(), req.GetApplicationId(), authPayload.Username)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "确认猎人失败: %v", err)
	}

	// 返回最新状态
	bounty, err := server.store.GetBountyByID(ctx, req.GetBountyId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "查询悬赏详情失败: %v", err)
	}
	return &pb.ConfirmHunterResponse{Bounty: convertBounty(bounty)}, nil
}
