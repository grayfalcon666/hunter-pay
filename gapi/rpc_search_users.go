package gapi

import (
	"context"

	"github.com/grayfalcon666/user-profile-service/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (server *Server) SearchUsers(ctx context.Context, req *pb.SearchUsersRequest) (*pb.SearchUsersResponse, error) {
	query := req.GetQuery()
	if query == "" {
		return nil, status.Errorf(codes.InvalidArgument, "搜索关键词不能为空")
	}

	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 10
	}

	profiles, err := server.store.SearchUsers(ctx, query, limit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "搜索用户失败: %v", err)
	}

	var users []*pb.UserSummary
	for _, p := range profiles {
		users = append(users, &pb.UserSummary{Username: p.Username})
	}

	return &pb.SearchUsersResponse{Users: users}, nil
}
