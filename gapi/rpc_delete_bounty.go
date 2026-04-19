package gapi

import (
	"context"

	"github.com/grayfalcon666/escrow-bounty/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (server *Server) DeleteBounty(ctx context.Context, req *pb.DeleteBountyRequest) (*pb.DeleteBountyResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}

	err = server.store.DeleteBounty(ctx, req.GetBountyId(), authPayload.Username)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "删除悬赏失败: %v", err)
	}

	return &pb.DeleteBountyResponse{
		Success: true,
	}, nil
}
