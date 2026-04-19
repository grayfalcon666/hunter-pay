package gapi

import (
	db "simplebank/db/sqlc"
	"simplebank/mq"
	"simplebank/pb"
	"simplebank/token"
	"simplebank/util"
	"simplebank/worker"
)

type Server struct {
	pb.UnimplementedSimpleBankServer
	config          util.Config
	store           db.Store
	tokenMaker      token.Maker
	taskDistributor worker.TaskDistributor
	userProducer    *mq.UserProducer
}

func NewServer(config util.Config, store db.Store, taskDistributor worker.TaskDistributor) (*Server, error) {
	// tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	tokenMaker, err := token.NewJWTMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, err
	}

	server := &Server{
		config:          config,
		store:           store,
		tokenMaker:      tokenMaker,
		taskDistributor: taskDistributor,
	}

	return server, nil
}

// SetUserProducer 设置用户事件生产者
func (s *Server) SetUserProducer(producer *mq.UserProducer) {
	s.userProducer = producer
}
