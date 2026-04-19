package gapi

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	mockdb "github.com/grayfalcon666/escrow-bounty/db/mock"
	"github.com/grayfalcon666/escrow-bounty/models"
	"github.com/grayfalcon666/escrow-bounty/pb"
	"github.com/grayfalcon666/escrow-bounty/token"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestListBountiesAPI(t *testing.T) {
	username := "alice"
	statusFilter := string(models.BountyStatusPending)

	req := &pb.ListBountiesRequest{
		Status:   statusFilter,
		PageId:   1,
		PageSize: 5,
	}

	mockBounties := []models.Bounty{
		{ID: 1, EmployerUsername: "bob", Status: models.BountyStatusPending},
		{ID: 2, EmployerUsername: "charlie", Status: models.BountyStatusPending},
	}

	testCases := []struct {
		name          string
		setupAuth     func(t *testing.T, requestCtx context.Context, tokenMaker *token.JWTMaker) context.Context
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, res *pb.ListBountiesResponse, err error)
	}{
		{
			name: "OK_获取列表成功",
			setupAuth: func(t *testing.T, requestCtx context.Context, tokenMaker *token.JWTMaker) context.Context {
				accessToken, _ := tokenMaker.CreateToken(username, time.Minute)
				md := metadata.MD{"authorization": []string{"Bearer " + accessToken}}
				return metadata.NewIncomingContext(requestCtx, md)
			},
			buildStubs: func(store *mockdb.MockStore) {
				// 期望传入的状态过滤、limit(5) 和 offset(0)
				store.EXPECT().
					ListBounties(gomock.Any(), models.BountyStatus(statusFilter), 5, 0).
					Times(1).
					Return(mockBounties, nil)
			},
			checkResponse: func(t *testing.T, res *pb.ListBountiesResponse, err error) {
				require.NoError(t, err)
				require.NotNil(t, res)
				require.Len(t, res.Bounties, 2)
			},
		},
		{
			name: "InternalError_数据库异常",
			setupAuth: func(t *testing.T, requestCtx context.Context, tokenMaker *token.JWTMaker) context.Context {
				accessToken, _ := tokenMaker.CreateToken(username, time.Minute)
				md := metadata.MD{"authorization": []string{"Bearer " + accessToken}}
				return metadata.NewIncomingContext(requestCtx, md)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListBounties(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1).
					Return(nil, fmt.Errorf("数据库连接断开"))
			},
			checkResponse: func(t *testing.T, res *pb.ListBountiesResponse, err error) {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, codes.Internal, st.Code())
			},
		},
		{
			name: "Unauthenticated_未登录",
			setupAuth: func(t *testing.T, requestCtx context.Context, tokenMaker *token.JWTMaker) context.Context {
				return requestCtx
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListBounties(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, res *pb.ListBountiesResponse, err error) {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, codes.Unauthenticated, st.Code())
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			bankClient := mockdb.NewMockBankClient(ctrl)
			tc.buildStubs(store)

			server := newTestServer(t, store, bankClient)
			ctx := tc.setupAuth(t, context.Background(), server.tokenMaker)

			res, err := server.ListBounties(ctx, req)
			tc.checkResponse(t, res, err)
		})
	}
}
