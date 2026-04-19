package gapi

import (
	"context"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

type Metadata struct {
	UserAgent string
	ClientIp  string
}

const (
	xForwardedHost       = "x-forwarded-host"
	UserAgent            = "user-agent"
	grpcGatewayUserAgent = "grpcgateway-user-agent"
)

func (server *Server) extractMetadata(ctx context.Context) *Metadata {
	mtdt := &Metadata{}

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if userAgents := md.Get(grpcGatewayUserAgent); len(userAgents) > 0 {
			mtdt.UserAgent = userAgents[0]
		}

		if userAgents := md.Get(UserAgent); len(userAgents) > 0 {
			mtdt.UserAgent = userAgents[0]
		}

		if clientIp := md.Get(xForwardedHost); len(clientIp) > 0 {
			mtdt.ClientIp = clientIp[0]
		}
	}

	if p, ok := peer.FromContext(ctx); ok {
		mtdt.ClientIp = p.Addr.String()
	}
	return mtdt
}
