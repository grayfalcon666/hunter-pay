package gapi

import (
	"github.com/grayfalcon666/escrow-bounty/db"
	"github.com/grayfalcon666/escrow-bounty/pb"
	"github.com/grayfalcon666/escrow-bounty/token"
	"github.com/grayfalcon666/escrow-bounty/util"
	"github.com/grayfalcon666/escrow-bounty/wshub"
)

type Server struct {
	pb.UnimplementedEscrowBountyServiceServer
	config        util.Config
	tokenMaker    *token.JWTMaker
	store         db.Store
	bankClient    db.BankClient
	profileClient db.ProfileClient
	wsHub         *wshub.Hub
}

func NewServer(config util.Config, store db.Store, bankClient db.BankClient, tokenMaker *token.JWTMaker, profileClient db.ProfileClient, wsHub *wshub.Hub) *Server {
	return &Server{
		config:        config,
		store:         store,
		bankClient:    bankClient,
		tokenMaker:    tokenMaker,
		profileClient: profileClient,
		wsHub:         wsHub,
	}
}
