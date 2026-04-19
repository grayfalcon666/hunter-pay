package gapi

import (
	"context"
	"fmt"
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

func TestUpdateProfile(t *testing.T) {
	secretKey := "12345678901234567890123456789012"

	testCases := []struct {
		name          string
		setupAuth     func() context.Context
		setupRequest  func() *pb.UpdateProfileRequest
		buildStubs    func(ctrl *gomock.Controller, store *mock.MockStore)
		checkResponse func(t *testing.T, res *pb.UpdateProfileResponse, err error)
	}{
		{
			name: "OK_更新成功",
			setupAuth: func() context.Context {
				tokenStr, _, _ := createUserToken(secretKey, "alice", time.Hour)
				return metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+tokenStr))
			},
			setupRequest: func() *pb.UpdateProfileRequest {
				return &pb.UpdateProfileRequest{
					Username:        "alice",
					ExpectedSalaryMin: "60000",
					WorkLocation:   "Beijing",
					ExperienceLevel: "SENIOR",
				}
			},
			buildStubs: func(ctrl *gomock.Controller, store *mock.MockStore) {
				store.EXPECT().
					UpdateProfile(gomock.Any(), "alice", gomock.Any()).
					Return(&models.UserProfile{
						Username:          "alice",
						ExpectedSalaryMin: "60000",
						WorkLocation:     "Beijing",
						ExperienceLevel:  models.ExperienceSenior,
					}, nil)
			},
			checkResponse: func(t *testing.T, res *pb.UpdateProfileResponse, err error) {
				require.NoError(t, err)
				require.Equal(t, "Beijing", res.Profile.WorkLocation)
				require.Equal(t, "SENIOR", res.Profile.ExperienceLevel)
			},
		},
		{
			name: "InternalError_存储失败",
			setupAuth: func() context.Context {
				tokenStr, _, _ := createUserToken(secretKey, "alice", time.Hour)
				return metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+tokenStr))
			},
			setupRequest: func() *pb.UpdateProfileRequest {
				return &pb.UpdateProfileRequest{
					Username: "alice",
					Bio:      "New bio",
				}
			},
			buildStubs: func(ctrl *gomock.Controller, store *mock.MockStore) {
				store.EXPECT().
					UpdateProfile(gomock.Any(), "alice", gomock.Any()).
					Return(nil, fmt.Errorf("数据库连接异常"))
			},
			checkResponse: func(t *testing.T, res *pb.UpdateProfileResponse, err error) {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, codes.Internal, st.Code())
			},
		},
		{
			name: "PermissionDenied_IDOR越权",
			setupAuth: func() context.Context {
				tokenStr, _, _ := createUserToken(secretKey, "alice", time.Hour)
				return metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+tokenStr))
			},
			setupRequest: func() *pb.UpdateProfileRequest {
				return &pb.UpdateProfileRequest{
					Username:    "bob",
					WorkLocation: "Remote",
				}
			},
			buildStubs: func(ctrl *gomock.Controller, store *mock.MockStore) {
				store.EXPECT().UpdateProfile(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, res *pb.UpdateProfileResponse, err error) {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, codes.PermissionDenied, st.Code())
			},
		},
		{
			name: "Unauthorized_未登录",
			setupAuth: func() context.Context {
				return context.Background()
			},
			setupRequest: func() *pb.UpdateProfileRequest {
				return &pb.UpdateProfileRequest{
					Username:    "alice",
					WorkLocation: "Remote",
				}
			},
			buildStubs: func(ctrl *gomock.Controller, store *mock.MockStore) {
				store.EXPECT().UpdateProfile(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, res *pb.UpdateProfileResponse, err error) {
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
			res, err := server.UpdateProfile(tc.setupAuth(), tc.setupRequest())
			tc.checkResponse(t, res, err)
		})
	}
}
