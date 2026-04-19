package gapi

import (
	"github.com/grayfalcon666/user-profile-service/models"
	"github.com/grayfalcon666/user-profile-service/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func convertProfile(m *models.UserProfile) *pb.UserProfile {
	return &pb.UserProfile{
		Username:                m.Username,
		ExpectedSalaryMin:      m.ExpectedSalaryMin,
		ExpectedSalaryMax:      m.ExpectedSalaryMax,
		WorkLocation:           m.WorkLocation,
		ExperienceLevel:        string(m.ExperienceLevel),
		Bio:                    m.Bio,
		AvatarUrl:              m.AvatarURL,
		CompletionRate:         m.CompletionRate,
		GoodReviewRate:          m.GoodReviewRate,
		TotalBountiesPosted:             int32(m.TotalBountiesPosted),
		TotalBountiesCompleted:          int32(m.TotalBountiesCompleted),
		TotalBountiesCompletedAsEmployer: int32(m.TotalBountiesCompletedAsEmployer),
		TotalEarnings:                   m.TotalEarnings,
		ReputationScore:        m.ReputationScore,
		Role:                   m.Role,
		CreatedAt:              timestamppb.New(m.CreatedAt),
		UpdatedAt:              timestamppb.New(m.UpdatedAt),
		HunterFulfillmentIndex:   int32(m.HunterFulfillmentIndex),
		EmployerFulfillmentIndex: int32(m.EmployerFulfillmentIndex),
	}
}

func convertReview(m *models.UserReview) *pb.UserReview {
	return &pb.UserReview{
		Id:               m.ID,
		ReviewerUsername: m.ReviewerUsername,
		ReviewedUsername: m.ReviewedUsername,
		BountyId:         m.BountyID,
		Rating:           int32(m.Rating),
		Comment:          m.Comment,
		ReviewType:       string(m.ReviewType),
		CreatedAt:        timestamppb.New(m.CreatedAt),
	}
}

func convertReviewSlice(reviews []models.UserReview) []*pb.UserReview {
	result := make([]*pb.UserReview, len(reviews))
	for i, r := range reviews {
		result[i] = convertReview(&r)
	}
	return result
}
