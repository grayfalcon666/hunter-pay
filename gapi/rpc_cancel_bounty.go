package gapi

import (
	"context"

	"github.com/grayfalcon666/escrow-bounty/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (server *Server) CancelBounty(ctx context.Context, req *pb.CancelBountyRequest) (*pb.CancelBountyResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}

	err = server.store.CancelBounty(ctx, req.GetBountyId(), authPayload.Username, server.bankClient)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "取消悬赏失败: %v", err)
	}

	bounty, err := server.store.GetBountyByID(ctx, req.GetBountyId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "查询悬赏详情失败: %v", err)
	}

	return &pb.CancelBountyResponse{Bounty: convertBounty(bounty)}, nil
}
