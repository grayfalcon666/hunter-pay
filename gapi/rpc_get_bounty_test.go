package gapi

import (
	"context"
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
	"gorm.io/gorm"
)

func TestGetBountyAPI(t *testing.T) {
	username := "alice"
	bountyID := int64(1)

	req := &pb.GetBountyRequest{
		Id: bountyID,
	}

	mockBounty := &models.Bounty{
		ID:               bountyID,
		EmployerUsername: "charlie",
		Title:            "测试悬赏",
		Applications: []models.BountyApplication{
			{ID: 10, HunterUsername: "bob"},
		},
	}

	testCases := []struct {
		name          string
		setupAuth     func(t *testing.T, requestCtx context.Context, tokenMaker *token.JWTMaker) context.Context
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, res *pb.GetBountyResponse, err error)
	}{
		{
			name: "OK_获取成功",
			setupAuth: func(t *testing.T, requestCtx context.Context, tokenMaker *token.JWTMaker) context.Context {
				accessToken, _ := tokenMaker.CreateToken(username, time.Minute)
				md := metadata.MD{"authorization": []string{"Bearer " + accessToken}}
				return metadata.NewIncomingContext(requestCtx, md)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetBountyByID(gomock.Any(), bountyID).
					Times(1).
					Return(mockBounty, nil)
			},
			checkResponse: func(t *testing.T, res *pb.GetBountyResponse, err error) {
				require.NoError(t, err)
				require.NotNil(t, res)
				require.Equal(t, mockBounty.ID, res.Bounty.Id)
				require.Len(t, res.Applications, 1)
			},
		},
		{
			name: "NotFound_记录不存在",
			setupAuth: func(t *testing.T, requestCtx context.Context, tokenMaker *token.JWTMaker) context.Context {
				accessToken, _ := tokenMaker.CreateToken(username, time.Minute)
				md := metadata.MD{"authorization": []string{"Bearer " + accessToken}}
				return metadata.NewIncomingContext(requestCtx, md)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetBountyByID(gomock.Any(), bountyID).
					Times(1).
					Return(nil, gorm.ErrRecordNotFound) // 模拟 GORM 的没找到错误
			},
			checkResponse: func(t *testing.T, res *pb.GetBountyResponse, err error) {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, codes.NotFound, st.Code())
			},
		},
		{
			name: "Unauthenticated_未登录",
			setupAuth: func(t *testing.T, requestCtx context.Context, tokenMaker *token.JWTMaker) context.Context {
				return requestCtx
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetBountyByID(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, res *pb.GetBountyResponse, err error) {
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

			res, err := server.GetBounty(ctx, req)
			tc.checkResponse(t, res, err)
		})
	}
}
