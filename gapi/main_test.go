package gapi

import (
	"testing"
	"time"

	"github.com/grayfalcon666/user-profile-service/token"
	"github.com/grayfalcon666/user-profile-service/util"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T) *Server {
	config := util.Config{
		DBSource:           "postgresql://root:secret@localhost:5433/escrow_db?sslmode=disable",
		GRPCServerAddress:  "0.0.0.0:9098",
		HTTPServerAddress:  "0.0.0.0:8088",
		TokenSymmetricKey:  "12345678901234567890123456789012",
		EscrowBountyAddress: "localhost:9097",
	}

	secretKey := "12345678901234567890123456789012"
	tokenMaker, err := token.NewJWTMaker(secretKey)
	require.NoError(t, err)

	server := NewServer(config, nil, tokenMaker)
	return server
}

func createUserToken(secretKey, username string, duration time.Duration) (string, *token.Payload, error) {
	maker, err := token.NewJWTMaker(secretKey)
	if err != nil {
		return "", nil, err
	}
	tk, err := maker.CreateToken(username, duration)
	if err != nil {
		return "", nil, err
	}
	payload, err := maker.VerifyToken(tk)
	if err != nil {
		return "", nil, err
	}
	return tk, payload, nil
}
