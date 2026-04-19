package gapi

import (
	"context"
	"testing"
	"time"

	"github.com/grayfalcon666/escrow-bounty/token"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAuthorizeUser(t *testing.T) {
	secretKey := "12345678901234567890123456789012"
	maker, err := token.NewJWTMaker(secretKey)
	require.NoError(t, err)

	validToken, err := maker.CreateToken("alice", time.Hour)
	require.NoError(t, err)

	expiredToken, err := maker.CreateToken("alice", -time.Hour)
	require.NoError(t, err)

	dummyToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6ImFsaWNlIiwiZXhwIjoxfQ.dummysig"

	testCases := []struct {
		name      string
		authHeader string
		wantCode  codes.Code
	}{
		{
			name:      "OK_有效Token",
			authHeader: "Bearer " + validToken,
			wantCode:  codes.OK,
		},
		{
			name:      "Unauthenticated_无Authorization头",
			authHeader: "",
			wantCode:  codes.Unauthenticated,
		},
		{
			name:      "Unauthenticated_Bearer后无Token",
			authHeader: "Bearer",
			wantCode:  codes.Unauthenticated,
		},
		{
			name:      "Unauthenticated_Token格式无效",
			authHeader: "Bearer " + dummyToken,
			wantCode:  codes.Unauthenticated,
		},
		{
			name:      "Unauthenticated_Token已过期",
			authHeader: "Bearer " + expiredToken,
			wantCode:  codes.Unauthenticated,
		},
		{
			name:      "Unauthenticated_不支持的认证类型",
			authHeader: "Basic dXNlcjpwYXNz",
			wantCode:  codes.Unauthenticated,
		},
		{
			name:      "OK_Bearer小写也通过（代码做了ToLower）",
			authHeader: "bearer " + validToken,
			wantCode:  codes.OK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ctx := context.Background()
			if tc.authHeader != "" {
				md := metadata.MD{"authorization": []string{tc.authHeader}}
				ctx = metadata.NewIncomingContext(ctx, md)
			}

			server := &Server{tokenMaker: maker}
			payload, err := server.authorizeUser(ctx)

			if tc.wantCode == codes.OK {
				require.NoError(t, err)
				require.NotNil(t, payload)
				require.Equal(t, "alice", payload.Username)
			} else {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, tc.wantCode, st.Code())
			}
		})
	}
}
