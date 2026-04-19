package gapi

import (
	"context"
	"errors"

	"github.com/grayfalcon666/escrow-bounty/models"
	"github.com/grayfalcon666/escrow-bounty/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func (server *Server) ListMessages(ctx context.Context, req *pb.ListMessagesRequest) (*pb.ListMessagesResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}
	username := authPayload.Username

	bountyID := req.GetBountyId()
	if bountyID <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "非法的悬赏 ID")
	}

	bounty, err := server.store.GetBountyByID(ctx, bountyID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "悬赏不存在")
		}
		return nil, status.Errorf(codes.Internal, "查询悬赏失败: %v", err)
	}

	// Verify the user is either the employer or the accepted hunter
	isEmployer := username == bounty.EmployerUsername
	isHunter := false
	for _, app := range bounty.Applications {
		if app.Status == models.AppStatusAccepted && app.HunterUsername == username {
			isHunter = true
			break
		}
	}
	if !isEmployer && !isHunter {
		return nil, status.Errorf(codes.PermissionDenied, "你不是该悬赏的雇主或中标猎人，无权查看消息")
	}

	msgs, err := server.store.ListMessages(ctx, bountyID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "查询消息失败: %v", err)
	}

	pbMsgs := make([]*pb.ChatMessage, len(msgs))
	for i := range msgs {
		pbMsgs[i] = convertChatMessage(&msgs[i], bountyID)
	}
	return &pb.ListMessagesResponse{Messages: pbMsgs}, nil
}
