package gapi

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	mockdb "github.com/grayfalcon666/escrow-bounty/db/mock"
	"github.com/grayfalcon666/escrow-bounty/pb"
	"github.com/grayfalcon666/escrow-bounty/token"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestCreateBountyAPI(t *testing.T) {
	employerUsername := "alice"
	employerAccountID := int64(10)
	rewardAmount := int64(5000)

	req := &pb.CreateBountyRequest{
		Title:              "修复 Bug",
		Description:        "紧急",
		RewardAmount:       rewardAmount,
		EmployerAccountId: employerAccountID,
	}

	testCases := []struct {
		name          string
		setupAuth     func(t *testing.T, requestCtx context.Context, tokenMaker *token.JWTMaker) context.Context
		setupRequest  func() *pb.CreateBountyRequest
		buildStubs    func(store *mockdb.MockStore, bankClient *mockdb.MockBankClient)
		checkResponse func(t *testing.T, res *pb.CreateBountyResponse, err error)
	}{
		{
			name: "OK_发布成功",
			setupAuth: func(t *testing.T, requestCtx context.Context, tokenMaker *token.JWTMaker) context.Context {
				accessToken, _ := tokenMaker.CreateToken(employerUsername, time.Minute)
				md := metadata.MD{"authorization": []string{"Bearer " + accessToken}}
				return metadata.NewIncomingContext(requestCtx, md)
			},
			setupRequest: func() *pb.CreateBountyRequest { return req },
			buildStubs: func(store *mockdb.MockStore, bankClient *mockdb.MockBankClient) {
				store.EXPECT().
					PublishBounty(gomock.Any(), gomock.Any(), gomock.Any(), employerAccountID, gomock.Any()).
					Times(1).
					Return(nil)
			},
			checkResponse: func(t *testing.T, res *pb.CreateBountyResponse, err error) {
				require.NoError(t, err)
				require.NotNil(t, res)
			},
		},
		{
			name: "InvalidArgument_悬赏金额为0",
			setupAuth: func(t *testing.T, requestCtx context.Context, tokenMaker *token.JWTMaker) context.Context {
				accessToken, _ := tokenMaker.CreateToken(employerUsername, time.Minute)
				md := metadata.MD{"authorization": []string{"Bearer " + accessToken}}
				return metadata.NewIncomingContext(requestCtx, md)
			},
			setupRequest: func() *pb.CreateBountyRequest {
				return &pb.CreateBountyRequest{
					Title:              "无奖励悬赏",
					RewardAmount:       0,
					EmployerAccountId:  employerAccountID,
				}
			},
			buildStubs: func(store *mockdb.MockStore, bankClient *mockdb.MockBankClient) {
				store.EXPECT().PublishBounty(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, res *pb.CreateBountyResponse, err error) {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, codes.InvalidArgument, st.Code())
			},
		},
		{
			name: "InvalidArgument_支付账户ID无效",
			setupAuth: func(t *testing.T, requestCtx context.Context, tokenMaker *token.JWTMaker) context.Context {
				accessToken, _ := tokenMaker.CreateToken(employerUsername, time.Minute)
				md := metadata.MD{"authorization": []string{"Bearer " + accessToken}}
				return metadata.NewIncomingContext(requestCtx, md)
			},
			setupRequest: func() *pb.CreateBountyRequest {
				return &pb.CreateBountyRequest{
					Title:              "无效账户悬赏",
					RewardAmount:       1000,
					EmployerAccountId:  0,
				}
			},
			buildStubs: func(store *mockdb.MockStore, bankClient *mockdb.MockBankClient) {
				store.EXPECT().PublishBounty(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, res *pb.CreateBountyResponse, err error) {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, codes.InvalidArgument, st.Code())
			},
		},
		{
			name: "InternalError_SimpleBank扣款失败",
			setupAuth: func(t *testing.T, requestCtx context.Context, tokenMaker *token.JWTMaker) context.Context {
				accessToken, _ := tokenMaker.CreateToken(employerUsername, time.Minute)
				md := metadata.MD{"authorization": []string{"Bearer " + accessToken}}
				return metadata.NewIncomingContext(requestCtx, md)
			},
			setupRequest: func() *pb.CreateBountyRequest { return req },
			buildStubs: func(store *mockdb.MockStore, bankClient *mockdb.MockBankClient) {
				store.EXPECT().
					PublishBounty(gomock.Any(), gomock.Any(), gomock.Any(), employerAccountID, gomock.Any()).
					Times(1).
					Return(fmt.Errorf("资金冻结失败"))
			},
			checkResponse: func(t *testing.T, res *pb.CreateBountyResponse, err error) {
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
			setupRequest: func() *pb.CreateBountyRequest { return req },
			buildStubs: func(store *mockdb.MockStore, bankClient *mockdb.MockBankClient) {
				store.EXPECT().PublishBounty(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, res *pb.CreateBountyResponse, err error) {
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
			tc.buildStubs(store, bankClient)

			server := newTestServer(t, store, bankClient)
			ctx := tc.setupAuth(t, context.Background(), server.tokenMaker)

			res, err := server.CreateBounty(ctx, tc.setupRequest())
			tc.checkResponse(t, res, err)
		})
	}
}
