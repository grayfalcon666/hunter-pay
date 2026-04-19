package gapi

import (
	"context"
	"strings"

	"github.com/grayfalcon666/escrow-bounty/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (server *Server) SubmitBounty(ctx context.Context, req *pb.SubmitBountyRequest) (*pb.SubmitBountyResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}

	bountyID := req.GetBountyId()
	if bountyID <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "无效的悬赏 ID")
	}

	submissionText := strings.TrimSpace(req.GetSubmissionText())
	if submissionText == "" {
		return nil, status.Errorf(codes.InvalidArgument, "提交内容不能为空")
	}
	if int64(len(submissionText)) > 5000 {
		return nil, status.Errorf(codes.InvalidArgument, "提交内容不能超过 5000 字符")
	}

	err = server.store.SubmitBounty(ctx, bountyID, authPayload.Username, submissionText)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "提交工作失败: %v", err)
	}

	bounty, err := server.store.GetBountyByID(ctx, bountyID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "查询悬赏详情失败: %v", err)
	}

	return &pb.SubmitBountyResponse{Bounty: convertBounty(bounty)}, nil
}
