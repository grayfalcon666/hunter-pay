package gapi

import (
	"context"

	"github.com/grayfalcon666/user-profile-service/pb"
)

func (server *Server) GetReviews(ctx context.Context, req *pb.GetReviewsRequest) (*pb.GetReviewsResponse, error) {
	_, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}

	reviews, err := server.store.GetReviewsByUser(ctx, req.Username)
	if err != nil {
		return nil, err
	}

	// 收集所有 reviewer，去重查头像
	usernameSet := make(map[string]struct{})
	for _, r := range reviews {
		usernameSet[r.ReviewerUsername] = struct{}{}
	}
	usernames := make([]string, 0, len(usernameSet))
	for u := range usernameSet {
		usernames = append(usernames, u)
	}
	avatarMap, _ := server.store.GetAvatarUrlsByUsernames(ctx, usernames)

	pbReviews := convertReviewSlice(reviews)
	for _, r := range pbReviews {
		r.ReviewerAvatarUrl = avatarMap[r.ReviewerUsername]
	}

	return &pb.GetReviewsResponse{
		Reviews: pbReviews,
	}, nil
}
