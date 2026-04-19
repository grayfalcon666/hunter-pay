package gapi

import (
	"context"

	"github.com/grayfalcon666/user-profile-service/models"
	"github.com/grayfalcon666/user-profile-service/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// InitializeProfile 系统调用：初始化用户资料（用于用户注册后自动创建资料）
// 使用 authorizeAdmin 进行系统认证（通过系统令牌）
func (server *Server) InitializeProfile(ctx context.Context, req *pb.InitializeProfileRequest) (*pb.InitializeProfileResponse, error) {
	// 系统调用，使用 authorizeAdmin 进行认证
	_, err := server.authorizeAdmin(ctx)
	if err != nil {
		return nil, err
	}

	// 幂等性检查：如果已经通过相同的 request_id 初始化过，直接返回
	existing, err := server.store.GetProfileByInitRequestID(ctx, req.RequestId)
	if err == nil && existing != "" {
		// 已经存在，返回现有资料
		profile, err := server.store.GetProfile(ctx, existing)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "获取用户资料失败：%v", err)
		}
		return &pb.InitializeProfileResponse{
			Profile:   convertProfile(profile),
			Created:   false,
			RequestId: req.RequestId,
		}, nil
	}

	// 检查用户资料是否已存在
	_, err = server.store.GetProfile(ctx, req.Username)
	if err == nil {
		// 已存在，更新 initialization_request_id
		err = server.store.UpdateProfileInitRequestID(ctx, req.Username, req.RequestId)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "更新初始化请求 ID 失败：%v", err)
		}
		profile, _ := server.store.GetProfile(ctx, req.Username)
		return &pb.InitializeProfileResponse{
			Profile:   convertProfile(profile),
			Created:   false,
			RequestId: req.RequestId,
		}, nil
	}

	// 创建用户资料
	params := &models.CreateProfileParams{
		ExpectedSalaryMin: req.ExpectedSalaryMin,
		ExpectedSalaryMax: req.ExpectedSalaryMax,
		WorkLocation:      req.WorkLocation,
		Bio:               req.Bio,
		AvatarURL:         req.AvatarUrl,
	}

	if req.ExperienceLevel != "" {
		params.ExperienceLevel = models.ExperienceLevel(req.ExperienceLevel)
	} else {
		params.ExperienceLevel = models.ExperienceEntry
	}

	profile, err := server.store.CreateProfile(ctx, req.Username, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "创建用户资料失败：%v", err)
	}

	// 更新 initialization_request_id
	err = server.store.UpdateProfileInitRequestID(ctx, req.Username, req.RequestId)
	if err != nil {
		// 非致命错误，继续返回
		return &pb.InitializeProfileResponse{
			Profile:   convertProfile(profile),
			Created:   true,
			RequestId: req.RequestId,
		}, nil
	}

	return &pb.InitializeProfileResponse{
		Profile:   convertProfile(profile),
		Created:   true,
		RequestId: req.RequestId,
	}, nil
}
