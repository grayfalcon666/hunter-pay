package gapi

import (
	"context"

	"github.com/grayfalcon666/user-profile-service/models"
	"github.com/grayfalcon666/user-profile-service/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (server *Server) CreateProfile(ctx context.Context, req *pb.CreateProfileRequest) (*pb.CreateProfileResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}

	username := authPayload.Username

	params := &models.CreateProfileParams{
		ExpectedSalaryMin: req.ExpectedSalaryMin,
		ExpectedSalaryMax: req.ExpectedSalaryMax,
		WorkLocation:     req.WorkLocation,
		Bio:              req.Bio,
		AvatarURL:        req.AvatarUrl,
	}

	if req.ExperienceLevel != "" {
		params.ExperienceLevel = models.ExperienceLevel(req.ExperienceLevel)
	} else {
		params.ExperienceLevel = models.ExperienceEntry
	}

	profile, err := server.store.CreateProfile(ctx, username, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "创建画像失败: %v", err)
	}

	return &pb.CreateProfileResponse{
		Profile: convertProfile(profile),
	}, nil
}
