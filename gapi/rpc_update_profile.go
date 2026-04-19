package gapi

import (
	"context"

	"github.com/grayfalcon666/user-profile-service/models"
	"github.com/grayfalcon666/user-profile-service/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (server *Server) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.UpdateProfileResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}

	// IDOR 防御：只能修改自己的画像
	if authPayload.Username != req.Username {
		return nil, status.Errorf(codes.PermissionDenied, "无权修改他人的用户画像")
	}

	params := &models.UpdateProfileParams{
		ExpectedSalaryMin: req.ExpectedSalaryMin,
		ExpectedSalaryMax: req.ExpectedSalaryMax,
		WorkLocation:     req.WorkLocation,
		Bio:              req.Bio,
		AvatarURL:        req.AvatarUrl,
	}

	if req.ExperienceLevel != "" {
		params.ExperienceLevel = models.ExperienceLevel(req.ExperienceLevel)
	}

	profile, err := server.store.UpdateProfile(ctx, req.Username, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "更新画像失败: %v", err)
	}

	return &pb.UpdateProfileResponse{
		Profile: convertProfile(profile),
	}, nil
}
