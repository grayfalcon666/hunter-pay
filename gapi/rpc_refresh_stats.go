package gapi

import (
	"context"

	"github.com/grayfalcon666/user-profile-service/models"
	"github.com/grayfalcon666/user-profile-service/pb"
	"gorm.io/gorm"
)

func (server *Server) RefreshProfileStats(ctx context.Context, req *pb.RefreshProfileStatsRequest) (*pb.RefreshProfileStatsResponse, error) {
	// Admin/system 调用，不需要额外鉴权（escrow-bounty 使用系统账号调用）
	_ = server.config.EscrowBountyAddress

	params := &models.RefreshStatsParams{
		BountyID:                  req.BountyId,
		DeltaCompleted:            req.DeltaCompleted,
		DeltaEarnings:            req.DeltaEarnings,
		DeltaPosted:               req.DeltaPosted,
		DeltaCompletedAsEmployer:  req.DeltaCompletedAsEmployer,
	}

	// 如果用户画像不存在，先创建
	_, err := server.store.GetProfile(ctx, req.Username)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			_, err = server.store.CreateProfile(ctx, req.Username, &models.CreateProfileParams{})
			if err != nil {
				return nil, err
			}
		}
	}

	profile, err := server.store.RefreshStats(ctx, req.Username, params)
	if err != nil {
		return nil, err
	}

	return &pb.RefreshProfileStatsResponse{
		Profile: convertProfile(profile),
	}, nil
}
