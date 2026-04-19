package gapi

import (
	"context"

	"github.com/grayfalcon666/user-profile-service/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (server *Server) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}

	// IDOR 防御：只能查看自己的或任意用户（这里允许查看任意用户，供前端展示用户信息）
	_ = authPayload

	if req.Username == "" {
		return nil, status.Errorf(codes.InvalidArgument, "username 不能为空")
	}

	profile, err := server.store.GetProfile(ctx, req.Username)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "用户画像不存在: %s", req.Username)
	}

	return &pb.GetProfileResponse{
		Profile: convertProfile(profile),
	}, nil
}
