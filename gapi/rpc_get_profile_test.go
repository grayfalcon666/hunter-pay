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
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestGetProfile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mock.NewMockStore(ctrl)
	server := newTestServer(t)
	server.store = store

	secretKey := "12345678901234567890123456789012"
	tokenStr, _, err := createUserToken(secretKey, "alice", time.Hour)
	require.NoError(t, err)

	mockedProfile := &models.UserProfile{
		Username:              "alice",
		ExpectedSalaryMin:     "50000",
		ExpectedSalaryMax:     "80000",
		WorkLocation:          "Remote",
		ExperienceLevel:       models.ExperienceMid,
		Bio:                   "Go developer",
		CompletionRate:        0.85,
		GoodReviewRate:        0.9,
		TotalBountiesPosted:   10,
		TotalBountiesCompleted: 8,
		TotalEarnings:         800000,
	}

	store.EXPECT().
		GetProfile(gomock.Any(), "alice").
		Return(mockedProfile, nil)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+tokenStr))
	req := &pb.GetProfileRequest{Username: "alice"}
	resp, err := server.GetProfile(ctx, req)

	require.NoError(t, err)
	require.Equal(t, "alice", resp.Profile.Username)
	require.Equal(t, "Remote", resp.Profile.WorkLocation)
	require.Equal(t, float64(0.85), resp.Profile.CompletionRate)
}

func TestGetProfile_Unauthorized(t *testing.T) {
	server := newTestServer(t)
	ctx := context.Background()
	req := &pb.GetProfileRequest{Username: "alice"}
	_, err := server.GetProfile(ctx, req)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, "Unauthenticated", st.Code().String())
}

func TestGetProfile_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mock.NewMockStore(ctrl)
	server := newTestServer(t)
	server.store = store

	secretKey := "12345678901234567890123456789012"
	tokenStr, _, err := createUserToken(secretKey, "alice", time.Hour)
	require.NoError(t, err)

	store.EXPECT().
		GetProfile(gomock.Any(), "bob").
		Return(nil, status.Error(5, "NOT_FOUND"))

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+tokenStr))
	req := &pb.GetProfileRequest{Username: "bob"}
	_, err = server.GetProfile(ctx, req)
	require.Error(t, err)
}
