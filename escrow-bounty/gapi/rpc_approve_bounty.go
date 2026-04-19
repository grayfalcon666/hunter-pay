package gapi

import (
	"context"

	"github.com/grayfalcon666/escrow-bounty/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (server *Server) ApproveBounty(ctx context.Context, req *pb.ApproveBountyRequest) (*pb.ApproveBountyResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}

	bountyID := req.GetBountyId()
	if bountyID <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "无效的悬赏 ID")
	}

	err = server.store.ApproveBounty(ctx, bountyID, authPayload.Username, server.bankClient)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "审核通过失败: %v", err)
	}

	bounty, err := server.store.GetBountyByID(ctx, bountyID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "查询悬赏详情失败: %v", err)
	}

	return &pb.ApproveBountyResponse{Bounty: convertBounty(bounty)}, nil
}
