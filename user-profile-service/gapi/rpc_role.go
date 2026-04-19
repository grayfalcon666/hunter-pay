package gapi

import (
	"context"

	"github.com/grayfalcon666/user-profile-service/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (server *Server) UpdateRole(ctx context.Context, req *pb.UpdateRoleRequest) (*pb.UpdateRoleResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}

	if authPayload.Username != req.GetUsername() {
		return nil, status.Errorf(codes.PermissionDenied, "只能修改自己的角色")
	}

	newRole := req.GetRole()
	if newRole != "HUNTER" && newRole != "POSTER" {
		return nil, status.Errorf(codes.InvalidArgument, "角色必须是 HUNTER 或 POSTER")
	}

	profile, err := server.store.UpdateRole(ctx, req.GetUsername(), newRole)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "更新角色失败: %v", err)
	}

	return &pb.UpdateRoleResponse{
		Profile: convertProfile(profile),
	}, nil
}

func (server *Server) SearchHunters(ctx context.Context, req *pb.SearchHuntersRequest) (*pb.SearchHuntersResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}

	query := req.GetQuery()
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 20
	}
	sortBy := req.GetSortBy()

	hunters, err := server.store.SearchHunters(ctx, query, limit, sortBy, authPayload.Username)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "搜索猎人失败: %v", err)
	}

	result := make([]*pb.UserProfile, len(hunters))
	for i, h := range hunters {
		result[i] = convertProfile(&h)
	}

	return &pb.SearchHuntersResponse{
		Hunters: result,
	}, nil
}
