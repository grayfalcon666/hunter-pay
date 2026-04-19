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

func TestCreateProfile(t *testing.T) {
	secretKey := "12345678901234567890123456789012"
	tokenStr, _, err := createUserToken(secretKey, "alice", time.Hour)
	require.NoError(t, err)

	testCases := []struct {
		name          string
		setupAuth     func() context.Context
		setupRequest  func() *pb.CreateProfileRequest
		buildStubs    func(ctrl *gomock.Controller, store *mock.MockStore)
		checkResponse func(t *testing.T, res *pb.CreateProfileResponse, err error)
	}{
		{
			name: "OK_创建成功",
			setupAuth: func() context.Context {
				return metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+tokenStr))
			},
			setupRequest: func() *pb.CreateProfileRequest {
				return &pb.CreateProfileRequest{
					ExpectedSalaryMin: "50000",
					ExpectedSalaryMax: "80000",
					WorkLocation:     "Remote",
					ExperienceLevel: "MID",
					Bio:             "Go dev",
				}
			},
			buildStubs: func(ctrl *gomock.Controller, store *mock.MockStore) {
				store.EXPECT().
					CreateProfile(gomock.Any(), "alice", gomock.Any()).
					Return(&models.UserProfile{
						Username:          "alice",
						ExpectedSalaryMin: "50000",
						ExpectedSalaryMax: "80000",
						WorkLocation:     "Remote",
						ExperienceLevel:  models.ExperienceMid,
					}, nil)
			},
			checkResponse: func(t *testing.T, res *pb.CreateProfileResponse, err error) {
				require.NoError(t, err)
				require.Equal(t, "alice", res.Profile.Username)
				require.Equal(t, "Remote", res.Profile.WorkLocation)
			},
		},
		{
			name: "InternalError_存储失败",
			setupAuth: func() context.Context {
				return metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+tokenStr))
			},
			setupRequest: func() *pb.CreateProfileRequest {
				return &pb.CreateProfileRequest{
					ExpectedSalaryMin: "50000",
					WorkLocation:     "Remote",
					ExperienceLevel:  "MID",
				}
			},
			buildStubs: func(ctrl *gomock.Controller, store *mock.MockStore) {
				store.EXPECT().
					CreateProfile(gomock.Any(), "alice", gomock.Any()).
					Return(nil, fmt.Errorf("数据库连接异常"))
			},
			checkResponse: func(t *testing.T, res *pb.CreateProfileResponse, err error) {
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
			setupRequest: func() *pb.CreateProfileRequest {
				return &pb.CreateProfileRequest{}
			},
			buildStubs: func(ctrl *gomock.Controller, store *mock.MockStore) {
				store.EXPECT().CreateProfile(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, res *pb.CreateProfileResponse, err error) {
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
			res, err := server.CreateProfile(tc.setupAuth(), tc.setupRequest())
			tc.checkResponse(t, res, err)
		})
	}
}
