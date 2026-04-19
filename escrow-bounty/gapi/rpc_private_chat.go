package gapi

import (
	"context"
	"strings"

	"github.com/grayfalcon666/escrow-bounty/models"
	"github.com/grayfalcon666/escrow-bounty/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (server *Server) GetOrCreateConversation(ctx context.Context, req *pb.GetOrCreateConversationRequest) (*pb.GetOrCreateConversationResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}
	caller := authPayload.Username
	other := strings.TrimSpace(req.GetOtherUsername())
	if other == "" {
		return nil, status.Errorf(codes.InvalidArgument, "用户名不能为空")
	}
	if other == caller {
		return nil, status.Errorf(codes.InvalidArgument, "不能和自己聊天")
	}

	conv, err := server.store.GetOrCreatePrivateConversation(ctx, caller, other)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "创建会话失败: %v", err)
	}
	return &pb.GetOrCreateConversationResponse{
		Conversation: convertPrivateConversation(conv, caller),
	}, nil
}

func (server *Server) ListConversations(ctx context.Context, req *pb.ListConversationsRequest) (*pb.ListConversationsResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}
	caller := authPayload.Username

	convs, err := server.store.ListPrivateConversations(ctx, caller)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "查询会话列表失败: %v", err)
	}

	unreadCounts, err := server.store.GetUnreadCounts(ctx, caller)
	if err != nil {
		unreadCounts = make(map[int64]int)
	}

	var summaries []*pb.ConversationSummary
	var totalUnread int64
	for _, conv := range convs {
		// Get last message for this conversation
		msgs, err := server.store.ListMessagesV2(ctx, conv.ID, 1, 0)
		var lastMsg *models.AllMessage
		if err == nil && len(msgs) > 0 {
			lastMsg = &msgs[len(msgs)-1]
		}
		unread := unreadCounts[conv.ID]
		totalUnread += int64(unread)
		summaries = append(summaries, convertConversationSummary(&conv, lastMsg, unread, caller))
	}

	return &pb.ListConversationsResponse{
		Conversations: summaries,
		TotalUnread:   totalUnread,
	}, nil
}

func (server *Server) ListPrivateMessages(ctx context.Context, req *pb.ListPrivateMessagesRequest) (*pb.ListPrivateMessagesResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}
	caller := authPayload.Username
	convID := req.GetConversationId()
	if convID <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "无效的会话 ID")
	}

	// Verify caller is a participant
	convs, err := server.store.ListPrivateConversations(ctx, caller)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "查询会话失败: %v", err)
	}
	valid := false
	for _, c := range convs {
		if c.ID == convID {
			valid = true
			break
		}
	}
	if !valid {
		return nil, status.Errorf(codes.PermissionDenied, "无权访问此会话")
	}

	limit := int(req.GetLimit())
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset := 0
	if req.GetBeforeId() > 0 {
		// Find offset based on before_id
		allMsgs, err := server.store.ListMessagesV2(ctx, convID, 1000, 0)
		if err == nil {
			for i, m := range allMsgs {
				if m.ID == req.GetBeforeId() {
					offset = i + 1
					break
				}
			}
		}
	}

	msgs, err := server.store.ListMessagesV2(ctx, convID, limit, offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "查询消息失败: %v", err)
	}

	var pbMsgs []*pb.PrivateMessage
	for _, m := range msgs {
		pbMsgs = append(pbMsgs, convertPrivateMessage(&m))
	}
	hasMore := len(msgs) == limit

	return &pb.ListPrivateMessagesResponse{
		Messages: pbMsgs,
		HasMore:  hasMore,
	}, nil
}

func (server *Server) DeleteConversation(ctx context.Context, req *pb.DeleteConversationRequest) (*pb.DeleteConversationResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}
	caller := authPayload.Username
	convID := req.GetConversationId()
	if convID <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "无效的会话 ID")
	}

	if err := server.store.DeletePrivateConversation(ctx, convID, caller); err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			return nil, status.Errorf(codes.PermissionDenied, "无权删除此会话")
		}
		return nil, status.Errorf(codes.Internal, "删除会话失败: %v", err)
	}
	return &pb.DeleteConversationResponse{}, nil
}

func (server *Server) MarkMessagesRead(ctx context.Context, req *pb.MarkReadRequest) (*pb.MarkReadResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}
	caller := authPayload.Username
	convID := req.GetConversationId()
	if convID <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "无效的会话 ID")
	}

	if err := server.store.MarkMessagesRead(ctx, convID, caller); err != nil {
		return nil, status.Errorf(codes.Internal, "标记已读失败: %v", err)
	}
	return &pb.MarkReadResponse{}, nil
}
