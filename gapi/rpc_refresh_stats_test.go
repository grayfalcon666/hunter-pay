package gapi

import (
	"context"
	"testing"

	"github.com/grayfalcon666/user-profile-service/db/mock"
	"github.com/grayfalcon666/user-profile-service/models"
	"github.com/grayfalcon666/user-profile-service/pb"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func TestRefreshProfileStats(t *testing.T) {
	testCases := []struct {
		name          string
		setupRequest  func() *pb.RefreshProfileStatsRequest
		buildStubs    func(ctrl *gomock.Controller, store *mock.MockStore)
		checkResponse func(t *testing.T, res *pb.RefreshProfileStatsResponse, err error)
	}{
		{
			name: "OK_画像已存在_刷新完成数",
			setupRequest: func() *pb.RefreshProfileStatsRequest {
				return &pb.RefreshProfileStatsRequest{
					Username:       "hunter1",
					BountyId:       10,
					DeltaCompleted: 1,
					DeltaEarnings:  50000,
					DeltaPosted:    0,
				}
			},
			buildStubs: func(ctrl *gomock.Controller, store *mock.MockStore) {
				gomock.InOrder(
					store.EXPECT().
						GetProfile(gomock.Any(), "hunter1").
						Return(&models.UserProfile{
							Username:              "hunter1",
							TotalBountiesCompleted: 8,
							TotalEarnings:         400000,
						}, nil),
					store.EXPECT().
						RefreshStats(gomock.Any(), "hunter1", gomock.Any()).
						Return(&models.UserProfile{
							Username:               "hunter1",
							TotalBountiesCompleted: 9,
							TotalEarnings:          450000,
							CompletionRate:         0.9,
						}, nil),
				)
			},
			checkResponse: func(t *testing.T, res *pb.RefreshProfileStatsResponse, err error) {
				require.NoError(t, err)
				require.Equal(t, "hunter1", res.Profile.Username)
				require.Equal(t, int32(9), res.Profile.TotalBountiesCompleted)
				require.Equal(t, int64(450000), res.Profile.TotalEarnings)
			},
		},
		{
			name: "OK_画像不存在_自动创建后刷新",
			setupRequest: func() *pb.RefreshProfileStatsRequest {
				return &pb.RefreshProfileStatsRequest{
					Username:       "newuser",
					BountyId:       1,
					DeltaPosted:    1,
					DeltaCompleted: 0,
					DeltaEarnings:  0,
				}
			},
			buildStubs: func(ctrl *gomock.Controller, store *mock.MockStore) {
				gomock.InOrder(
					store.EXPECT().
						GetProfile(gomock.Any(), "newuser").
						Return(nil, gorm.ErrRecordNotFound),
					store.EXPECT().
						CreateProfile(gomock.Any(), "newuser", gomock.Any()).
						Return(&models.UserProfile{Username: "newuser"}, nil),
					store.EXPECT().
						RefreshStats(gomock.Any(), "newuser", gomock.Any()).
						Return(&models.UserProfile{
							Username:             "newuser",
							TotalBountiesPosted:  1,
							TotalBountiesCompleted: 0,
						}, nil),
				)
			},
			checkResponse: func(t *testing.T, res *pb.RefreshProfileStatsResponse, err error) {
				require.NoError(t, err)
				require.Equal(t, "newuser", res.Profile.Username)
			},
		},
		{
			name: "InternalError_刷新统计存储失败",
			setupRequest: func() *pb.RefreshProfileStatsRequest {
				return &pb.RefreshProfileStatsRequest{
					Username:       "hunter1",
					DeltaCompleted: 1,
				}
			},
			buildStubs: func(ctrl *gomock.Controller, store *mock.MockStore) {
				gomock.InOrder(
					store.EXPECT().
						GetProfile(gomock.Any(), "hunter1").
						Return(&models.UserProfile{Username: "hunter1"}, nil),
					store.EXPECT().
						RefreshStats(gomock.Any(), "hunter1", gomock.Any()).
						Return(nil, status.Error(codes.Internal, "数据库异常")),
				)
			},
			checkResponse: func(t *testing.T, res *pb.RefreshProfileStatsResponse, err error) {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, codes.Internal, st.Code())
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
			res, err := server.RefreshProfileStats(context.Background(), tc.setupRequest())
			tc.checkResponse(t, res, err)
		})
	}
}
