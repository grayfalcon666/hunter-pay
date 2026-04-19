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

	return &pb.GetReviewsResponse{
		Reviews: convertReviewSlice(reviews),
	}, nil
}
