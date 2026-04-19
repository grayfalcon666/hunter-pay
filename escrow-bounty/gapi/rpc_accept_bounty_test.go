package gapi

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	mockdb "github.com/grayfalcon666/escrow-bounty/db/mock"
	"github.com/grayfalcon666/escrow-bounty/models"
	"github.com/grayfalcon666/escrow-bounty/pb"
	"github.com/grayfalcon666/escrow-bounty/token"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAcceptBountyAPI(t *testing.T) {
	hunterUsername := "bob"
	bountyID := int64(1)
	hunterAccountID := int64(20)

	req := &pb.AcceptBountyRequest{
		BountyId:        bountyID,
		HunterAccountId: hunterAccountID,
	}

	mockApplication := &models.BountyApplication{
		ID:              100,
		BountyID:        bountyID,
		HunterUsername:  hunterUsername,
		HunterAccountID: hunterAccountID,
		Status:          models.AppStatusApplied,
	}

	testCases := []struct {
		name          string
		req           *pb.AcceptBountyRequest
		setupAuth     func(t *testing.T, requestCtx context.Context, tokenMaker *token.JWTMaker) context.Context
		buildStubs    func(store *mockdb.MockStore, bankClient *mockdb.MockBankClient)
		checkResponse func(t *testing.T, res *pb.AcceptBountyResponse, err error)
	}{
		{
			name: "OK_接单成功",
			req:  req,
			setupAuth: func(t *testing.T, requestCtx context.Context, tokenMaker *token.JWTMaker) context.Context {
				accessToken, _ := tokenMaker.CreateToken(hunterUsername, time.Minute)
				md := metadata.MD{"authorization": []string{"Bearer " + accessToken}}
				return metadata.NewIncomingContext(requestCtx, md)
			},
			buildStubs: func(store *mockdb.MockStore, bankClient *mockdb.MockBankClient) {
				// 🔥 修复：添加对 VerifyAccountOwner 的预期，模拟校验通过 (返回 nil)
				bankClient.EXPECT().
					VerifyAccountOwner(gomock.Any(), hunterAccountID).
					Times(1).
					Return(nil)

				store.EXPECT().
					AcceptBounty(gomock.Any(), bountyID, hunterAccountID, hunterUsername).
					Times(1).
					Return(mockApplication, nil)
			},
			checkResponse: func(t *testing.T, res *pb.AcceptBountyResponse, err error) {
				require.NoError(t, err)
				require.NotNil(t, res)
				require.Equal(t, mockApplication.ID, res.Application.Id)
			},
		},
		{
			name: "AlreadyExists_重复接单",
			req:  req,
			setupAuth: func(t *testing.T, requestCtx context.Context, tokenMaker *token.JWTMaker) context.Context {
				accessToken, _ := tokenMaker.CreateToken(hunterUsername, time.Minute)
				md := metadata.MD{"authorization": []string{"Bearer " + accessToken}}
				return metadata.NewIncomingContext(requestCtx, md)
			},
			buildStubs: func(store *mockdb.MockStore, bankClient *mockdb.MockBankClient) {
				// 🔥 校验通过，但在写数据库时发生唯一索引冲突
				bankClient.EXPECT().
					VerifyAccountOwner(gomock.Any(), hunterAccountID).
					Times(1).
					Return(nil)

				store.EXPECT().
					AcceptBounty(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1).
					Return(nil, errors.New("duplicate key value violates unique constraint"))
			},
			checkResponse: func(t *testing.T, res *pb.AcceptBountyResponse, err error) {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, codes.AlreadyExists, st.Code())
			},
		},
		{
			name: "PermissionDenied_账号归属越权拦截",
			req:  req,
			setupAuth: func(t *testing.T, requestCtx context.Context, tokenMaker *token.JWTMaker) context.Context {
				accessToken, _ := tokenMaker.CreateToken(hunterUsername, time.Minute)
				md := metadata.MD{"authorization": []string{"Bearer " + accessToken}}
				return metadata.NewIncomingContext(requestCtx, md)
			},
			buildStubs: func(store *mockdb.MockStore, bankClient *mockdb.MockBankClient) {
				// 🔥 模拟 Simple Bank 返回账号不属于该猎人
				bankClient.EXPECT().
					VerifyAccountOwner(gomock.Any(), hunterAccountID).
					Times(1).
					Return(fmt.Errorf("账户归属校验失败"))

				// 因为被拦截，所以数据库的 AcceptBounty 绝对不会被调用 (Times: 0)
				store.EXPECT().AcceptBounty(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, res *pb.AcceptBountyResponse, err error) {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, codes.PermissionDenied, st.Code())
			},
		},
		{
			name: "InvalidArgument_参数错误",
			req: &pb.AcceptBountyRequest{
				BountyId: 0, // 非法 ID
			},
			setupAuth: func(t *testing.T, requestCtx context.Context, tokenMaker *token.JWTMaker) context.Context {
				accessToken, _ := tokenMaker.CreateToken(hunterUsername, time.Minute)
				md := metadata.MD{"authorization": []string{"Bearer " + accessToken}}
				return metadata.NewIncomingContext(requestCtx, md)
			},
			buildStubs: func(store *mockdb.MockStore, bankClient *mockdb.MockBankClient) {
				// 参数校验在最前面，不会调用外部服务和数据库
				bankClient.EXPECT().VerifyAccountOwner(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().AcceptBounty(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, res *pb.AcceptBountyResponse, err error) {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, codes.InvalidArgument, st.Code())
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
			tc.buildStubs(store, bankClient)

			server := newTestServer(t, store, bankClient)
			ctx := tc.setupAuth(t, context.Background(), server.tokenMaker)

			res, err := server.AcceptBounty(ctx, tc.req)
			tc.checkResponse(t, res, err)
		})
	}
}
