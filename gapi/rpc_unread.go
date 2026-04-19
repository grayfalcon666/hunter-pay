package gapi

import (
	"context"

	"github.com/grayfalcon666/escrow-bounty/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (server *Server) GetUnreadCounts(ctx context.Context, req *pb.GetUnreadCountsRequest) (*pb.GetUnreadCountsResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}
	caller := authPayload.Username

	counts, err := server.store.GetUnreadCounts(ctx, caller)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "查询未读数失败: %v", err)
	}

	pbCounts := make(map[int64]int32)
	var total int64
	for convID, count := range counts {
		pbCounts[convID] = int32(count)
		total += int64(count)
	}

	return &pb.GetUnreadCountsResponse{
		Counts: pbCounts,
		Total:  total,
	}, nil
}
