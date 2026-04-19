package gapi

import (
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func unauthenticatedError(err error) error {
	return status.Errorf(codes.Unauthenticated, "unauthenticated: %v", err)
}

// 权限不足的错误处理 HTTP 403
func permissionDeniedError(err error) error {
	return status.Errorf(codes.PermissionDenied, "permission denied: %v", err)
}

// 把一个字段名和错误信息包装成 gRPC 专用的详情对象
func fieldViolation(field string, err error) *errdetails.BadRequest_FieldViolation {
	return &errdetails.BadRequest_FieldViolation{
		Field:       field,
		Description: err.Error(),
	}
}

// 将多个字段错误打包成一个最终的 gRPC 状态
func invalidArgumentError(violations []*errdetails.BadRequest_FieldViolation) error {
	badRequest := &errdetails.BadRequest{FieldViolations: violations}

	statusInvalid := status.New(codes.InvalidArgument, "invalid parameters")

	// 向状态中挂载详情 (Details)
	statusWithDetails, err := statusInvalid.WithDetails(badRequest)
	if err != nil {
		return statusInvalid.Err()
	}

	return statusWithDetails.Err()
}
