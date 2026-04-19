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

func TestConfirmHunterAPI(t *testing.T) {
	employerUsername := "alice"
	bountyID := int64(1)
	applicationID := int64(100)

	req := &pb.ConfirmHunterRequest{
		BountyId:      bountyID,
		ApplicationId: applicationID,
	}

	mockBountyDetail := &models.Bounty{
		ID:               bountyID,
		EmployerUsername: employerUsername,
		Status:           models.BountyStatusInProgress,
		Applications: []models.BountyApplication{
			{ID: 100, Status: models.AppStatusAccepted, HunterUsername: "bob"},
			{ID: 101, Status: models.AppStatusRejected, HunterUsername: "charlie"},
		},
	}

	testCases := []struct {
		name          string
		setupAuth     func(t *testing.T, requestCtx context.Context, tokenMaker *token.JWTMaker) context.Context
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, res *pb.ConfirmHunterResponse, err error)
	}{
		{
			name: "OK_确认选定并返回最新状态",
			setupAuth: func(t *testing.T, requestCtx context.Context, tokenMaker *token.JWTMaker) context.Context {
				accessToken, _ := tokenMaker.CreateToken(employerUsername, time.Minute)
				md := metadata.MD{"authorization": []string{"Bearer " + accessToken}}
				return metadata.NewIncomingContext(requestCtx, md)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ConfirmHunter(gomock.Any(), bountyID, applicationID, employerUsername).
					Times(1).
					Return(nil)
				store.EXPECT().
					GetBountyByID(gomock.Any(), bountyID).
					Times(1).
					Return(mockBountyDetail, nil)
			},
			checkResponse: func(t *testing.T, res *pb.ConfirmHunterResponse, err error) {
				require.NoError(t, err)
				require.NotNil(t, res)
				require.Equal(t, string(models.BountyStatusInProgress), res.Bounty.Status)
			},
		},
		{
			name: "InternalError_悬赏不存在或权限不足",
			setupAuth: func(t *testing.T, requestCtx context.Context, tokenMaker *token.JWTMaker) context.Context {
				accessToken, _ := tokenMaker.CreateToken(employerUsername, time.Minute)
				md := metadata.MD{"authorization": []string{"Bearer " + accessToken}}
				return metadata.NewIncomingContext(requestCtx, md)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ConfirmHunter(gomock.Any(), bountyID, applicationID, employerUsername).
					Times(1).
					Return(fmt.Errorf("权限不足: 只能操作您自己发布的悬赏"))
				store.EXPECT().GetBountyByID(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, res *pb.ConfirmHunterResponse, err error) {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, codes.Internal, st.Code())
			},
		},
		{
			name: "InternalError_GetBountyByID失败",
			setupAuth: func(t *testing.T, requestCtx context.Context, tokenMaker *token.JWTMaker) context.Context {
				accessToken, _ := tokenMaker.CreateToken(employerUsername, time.Minute)
				md := metadata.MD{"authorization": []string{"Bearer " + accessToken}}
				return metadata.NewIncomingContext(requestCtx, md)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ConfirmHunter(gomock.Any(), bountyID, applicationID, employerUsername).
					Times(1).
					Return(nil)
				store.EXPECT().
					GetBountyByID(gomock.Any(), bountyID).
					Times(1).
					Return(nil, fmt.Errorf("数据库查询异常"))
			},
			checkResponse: func(t *testing.T, res *pb.ConfirmHunterResponse, err error) {
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
				store.EXPECT().ConfirmHunter(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, res *pb.ConfirmHunterResponse, err error) {
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

			res, err := server.ConfirmHunter(ctx, req)
			tc.checkResponse(t, res, err)
		})
	}
}
