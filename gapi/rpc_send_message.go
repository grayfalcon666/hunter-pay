package gapi

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/grayfalcon666/escrow-bounty/models"
	"github.com/grayfalcon666/escrow-bounty/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func (server *Server) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}
	username := authPayload.Username

	bountyID := req.GetBountyId()
	if bountyID <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "非法的悬赏 ID")
	}
	content := strings.TrimSpace(req.GetContent())
	if content == "" {
		return nil, status.Errorf(codes.InvalidArgument, "消息内容不能为空")
	}
	if int64(len(content)) > 2000 {
		return nil, status.Errorf(codes.InvalidArgument, "消息内容不能超过 2000 字符")
	}

	bounty, err := server.store.GetBountyByID(ctx, bountyID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "悬赏不存在")
		}
		return nil, status.Errorf(codes.Internal, "查询悬赏失败: %v", err)
	}

	if bounty.Status != models.BountyStatusInProgress {
		return nil, status.Errorf(codes.FailedPrecondition, "悬赏尚未开始或已结束，当前状态: %s", bounty.Status)
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
		return nil, status.Errorf(codes.PermissionDenied, "你不是该悬赏的雇主或中标猎人，无权发送消息")
	}

	chat, err := server.store.GetOrCreateChat(ctx, bountyID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "创建会话失败: %v", err)
	}

	msg, err := server.store.CreateMessage(ctx, chat.ID, username, content)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "发送消息失败: %v", err)
	}

	// Broadcast to other participants via WebSocket
	if server.wsHub != nil {
		go func() {
			payload, _ := json.Marshal(map[string]interface{}{
				"type": "new_message",
				"payload": map[string]interface{}{
					"id":               msg.ID,
					"bounty_chat_id":   chat.ID,
					"sender_username":   username,
					"content":          content,
					"is_read":          false,
					"created_at":       msg.CreatedAt.Unix(),
				},
			})
			server.wsHub.BroadcastToConversation(chat.ID, payload, nil)
		}()
	}

	return &pb.SendMessageResponse{
		Message: convertChatMessage(msg, bountyID),
	}, nil
}
