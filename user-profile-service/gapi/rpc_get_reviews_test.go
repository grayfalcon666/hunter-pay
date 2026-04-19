package gapi

import (
	"context"
	"testing"
	"time"

	"github.com/grayfalcon666/user-profile-service/db/mock"
	"github.com/grayfalcon666/user-profile-service/models"
	"github.com/grayfalcon666/user-profile-service/pb"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestGetReviews(t *testing.T) {
	secretKey := "12345678901234567890123456789012"

	testCases := []struct {
		name          string
		setupAuth     func() context.Context
		setupRequest  func() *pb.GetReviewsRequest
		buildStubs    func(ctrl *gomock.Controller, store *mock.MockStore)
		checkResponse func(t *testing.T, res *pb.GetReviewsResponse, err error)
	}{
		{
			name: "OK_获取成功",
			setupAuth: func() context.Context {
				tokenStr, _, _ := createUserToken(secretKey, "alice", time.Hour)
				return metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+tokenStr))
			},
			setupRequest: func() *pb.GetReviewsRequest {
				return &pb.GetReviewsRequest{Username: "hunter1"}
			},
			buildStubs: func(ctrl *gomock.Controller, store *mock.MockStore) {
				store.EXPECT().
					GetReviewsByUser(gomock.Any(), "hunter1").
					Return([]models.UserReview{
						{ID: 1, ReviewerUsername: "employer1", ReviewedUsername: "hunter1", BountyID: 10, Rating: 5, Comment: "Great!", ReviewType: models.ReviewEmployerToHunter},
						{ID: 2, ReviewerUsername: "employer2", ReviewedUsername: "hunter1", BountyID: 11, Rating: 4, Comment: "Good", ReviewType: models.ReviewEmployerToHunter},
					}, nil)
			},
			checkResponse: func(t *testing.T, res *pb.GetReviewsResponse, err error) {
				require.NoError(t, err)
				require.Len(t, res.Reviews, 2)
				require.Equal(t, int64(1), res.Reviews[0].Id)
				require.Equal(t, "employer1", res.Reviews[0].ReviewerUsername)
			},
		},
		{
			name: "OK_暂无评价",
			setupAuth: func() context.Context {
				tokenStr, _, _ := createUserToken(secretKey, "alice", time.Hour)
				return metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+tokenStr))
			},
			setupRequest: func() *pb.GetReviewsRequest {
				return &pb.GetReviewsRequest{Username: "newuser"}
			},
			buildStubs: func(ctrl *gomock.Controller, store *mock.MockStore) {
				store.EXPECT().
					GetReviewsByUser(gomock.Any(), "newuser").
					Return([]models.UserReview{}, nil)
			},
			checkResponse: func(t *testing.T, res *pb.GetReviewsResponse, err error) {
				require.NoError(t, err)
				require.Len(t, res.Reviews, 0)
			},
		},
		{
			name: "InternalError_数据库异常",
			setupAuth: func() context.Context {
				tokenStr, _, _ := createUserToken(secretKey, "alice", time.Hour)
				return metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+tokenStr))
			},
			setupRequest: func() *pb.GetReviewsRequest {
				return &pb.GetReviewsRequest{Username: "hunter1"}
			},
			buildStubs: func(ctrl *gomock.Controller, store *mock.MockStore) {
				store.EXPECT().
					GetReviewsByUser(gomock.Any(), "hunter1").
					Return(nil, status.Error(codes.Internal, "数据库连接异常"))
			},
			checkResponse: func(t *testing.T, res *pb.GetReviewsResponse, err error) {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, codes.Internal, st.Code())
			},
		},
		{
			name: "Unauthorized_未登录",
			setupAuth: func() context.Context {
				return context.Background()
			},
			setupRequest: func() *pb.GetReviewsRequest {
				return &pb.GetReviewsRequest{Username: "hunter1"}
			},
			buildStubs: func(ctrl *gomock.Controller, store *mock.MockStore) {
				store.EXPECT().GetReviewsByUser(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, res *pb.GetReviewsResponse, err error) {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, codes.Unauthenticated, st.Code())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			store := mock.NewMockStore(ctrl)
			server := newTestServer(t)
			server.store = store
			tc.buildStubs(ctrl, store)
			res, err := server.GetReviews(tc.setupAuth(), tc.setupRequest())
			tc.checkResponse(t, res, err)
		})
	}
}
