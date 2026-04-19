package gapi

import (
	"log"
	"testing"

	"github.com/grayfalcon666/escrow-bounty/db"
	"github.com/grayfalcon666/escrow-bounty/token"
	"github.com/grayfalcon666/escrow-bounty/util"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T, store db.Store, bankClient db.BankClient) *Server {
	config, err := util.LoadConfig("..")
	if err != nil {
		log.Fatalf("无法加载配置: %v", err)
	}

	secretKey := "12345678901234567890123456789012"
	tokenMaker, err := token.NewJWTMaker(secretKey)
	require.NoError(t, err)

	server := NewServer(config, store, bankClient, tokenMaker, nil, nil)
	return server
}
