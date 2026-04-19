package api

import (
	"fmt"
	"net/http"
	db "simplebank/db/sqlc"
	"simplebank/token"
	"simplebank/util"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func NewTestServer(t *testing.T, store db.Store) *Server {
	config := util.Config{
		TokenSymmetricKey:   util.RandomString(32),
		AccessTokenDuration: time.Minute,
	}

	server, err := NewServer(config, store)
	require.NoError(t, err)

	return server
}

func setupAuth(
	t *testing.T,
	request *http.Request,
	tokenMaker token.Maker,
	authorizationType string,
	username string,
	duration time.Duration,
) {
	token, payload, err := tokenMaker.CreateToken(username, duration)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NotEmpty(t, payload)

	// 构造 Header 值
	authorizationHeader := fmt.Sprintf("%s %s", authorizationType, token)
	request.Header.Set("authorization", authorizationHeader)
}
