package gapi

import (
	"context"
	"testing"
	"time"

	"github.com/grayfalcon666/user-profile-service/db/mock"
	"github.com/grayfalcon666/user-profile-service/models"
	"github.com/grayfalcon666/user-profile-service/pb"
	"go.uber.org/mock/gomock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestCreateReview_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mock.NewMockStore(ctrl)
	server := newTestServer(t)
	server.store = store

	secretKey := "12345678901234567890123456789012"
	tokenStr, _, err := createUserToken(secretKey, "employer1", time.Hour)
	require.NoError(t, err)

	mockedReview := &models.UserReview{
		ID:               1,
		ReviewerUsername:  "employer1",
		ReviewedUsername: "hunter1",
		BountyID:         10,
		Rating:           5,
		Comment:          "Great work!",
		ReviewType:       models.ReviewEmployerToHunter,
	}

	store.EXPECT().
		CreateReview(gomock.Any(), "employer1", gomock.Any()).
		Return(mockedReview, nil)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+tokenStr))
	req := &pb.CreateReviewRequest{
		ReviewedUsername: "hunter1",
		BountyId:         10,
		Rating:           5,
		Comment:          "Great work!",
		ReviewType:       "EMPLOYER_TO_HUNTER",
	}
	resp, err := server.CreateReview(ctx, req)

	require.NoError(t, err)
	require.Equal(t, int64(1), resp.Review.Id)
	require.Equal(t, "hunter1", resp.Review.ReviewedUsername)
	require.Equal(t, int32(5), resp.Review.Rating)
}

func TestCreateReview_SelfReview(t *testing.T) {
	server := newTestServer(t)

	secretKey := "12345678901234567890123456789012"
	tokenStr, _, err := createUserToken(secretKey, "alice", time.Hour)
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+tokenStr))
	req := &pb.CreateReviewRequest{
		ReviewedUsername: "alice",
		BountyId:         10,
		Rating:           5,
		ReviewType:       "EMPLOYER_TO_HUNTER",
	}
	_, err = server.CreateReview(ctx, req)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.InvalidArgument, st.Code())
}

func TestCreateReview_InvalidRating(t *testing.T) {
	server := newTestServer(t)

	secretKey := "12345678901234567890123456789012"
	tokenStr, _, err := createUserToken(secretKey, "alice", time.Hour)
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+tokenStr))
	req := &pb.CreateReviewRequest{
		ReviewedUsername: "bob",
		BountyId:         10,
		Rating:           6, // invalid
		ReviewType:       "EMPLOYER_TO_HUNTER",
	}
	_, err = server.CreateReview(ctx, req)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.InvalidArgument, st.Code())
}
