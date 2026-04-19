package gapi

import (
	"context"
	"fmt"
	"simplebank/token"
	"strings"

	"google.golang.org/grpc/metadata"
)

const (
	authorizationHeader = "authorization"
	authorizationBearer = "bearer"
)

const systemUsername = "escrow"

func (server *Server) authorizeUser(ctx context.Context) (*token.Payload, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("missing metadata")
	}

	v := md.Get(authorizationHeader)
	if len(v) == 0 {
		return nil, fmt.Errorf("missing authorization header")
	}

	authHeader := v[0]
	fields := strings.Fields(authHeader)
	if len(fields) < 2 {
		return nil, fmt.Errorf("invalid authorization")
	}

	authType := strings.ToLower(fields[0])
	if authType != authorizationBearer {
		return nil, fmt.Errorf("invalid authorization type %s, support %s only", authType, authorizationBearer)
	}

	accessToken := fields[1]
	payload, err := server.tokenMaker.VerifyToken(accessToken)
	if err != nil {
		return nil, fmt.Errorf("invalid access token %w", err)
	}

	// 系统账号 "escrow" 有权限操作任意账户，跳过账户归属检查
	if payload.Username == systemUsername {
		return payload, nil
	}

	return payload, nil
}
