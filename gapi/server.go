package gapi

import (
	"github.com/grayfalcon666/user-profile-service/db"
	"github.com/grayfalcon666/user-profile-service/mq"
	"github.com/grayfalcon666/user-profile-service/pb"
	"github.com/grayfalcon666/user-profile-service/token"
	"github.com/grayfalcon666/user-profile-service/util"
)

type Server struct {
	pb.UnimplementedProfileServiceServer
	config         util.Config
	tokenMaker     *token.JWTMaker
	store          db.Store
	eventProducer  *mq.UserEventProducer
}

func NewServer(config util.Config, store db.Store, tokenMaker *token.JWTMaker, eventProducer *mq.UserEventProducer) *Server {
	return &Server{
		config:        config,
		store:         store,
		tokenMaker:    tokenMaker,
		eventProducer: eventProducer,
	}
}
