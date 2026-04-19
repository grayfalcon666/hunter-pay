package gapi

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func invalidArgumentError(violations []*Violation) error {
	return status.Errorf(codes.InvalidArgument, "参数校验失败: %v", violations)
}

type Violation struct {
	Field  string
	Detail string
}
