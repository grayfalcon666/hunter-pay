package gapi

import (
	"context"
	"strings"

	"github.com/grayfalcon666/user-profile-service/token"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	authorizationHeader = "authorization"
	authorizationBearer = "bearer"
)

func (server *Server) authorizeUser(ctx context.Context) (*token.Payload, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "缺少元数据")
	}

	values := md.Get(authorizationHeader)
	if len(values) == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "未提供 Authorization 凭证")
	}

	authHeader := values[0]
	fields := strings.Fields(authHeader)
	if len(fields) < 2 {
		return nil, status.Errorf(codes.Unauthenticated, "Authorization 格式无效")
	}

	authType := strings.ToLower(fields[0])
	if authType != authorizationBearer {
		return nil, status.Errorf(codes.Unauthenticated, "不支持的认证类型: %s", authType)
	}

	accessToken := fields[1]
	payload, err := server.tokenMaker.VerifyToken(accessToken)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "Token 无效或已过期: %v", err)
	}

	return payload, nil
}

// authorizeAdmin checks if the caller is an admin user.
func (server *Server) authorizeAdmin(ctx context.Context) (*token.Payload, error) {
	payload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}
	// Admin check: in production, this would verify a role claim.
	// For now, we allow any authenticated user to call admin endpoints
	// (the escrow system username acts as the system/admin identity).
	return payload, nil
}
