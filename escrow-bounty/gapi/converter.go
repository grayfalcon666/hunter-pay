package gapi

import (
	"github.com/grayfalcon666/escrow-bounty/models"
	"github.com/grayfalcon666/escrow-bounty/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func convertBounty(bounty *models.Bounty) *pb.Bounty {
	pbBounty := &pb.Bounty{
		Id:                bounty.ID,
		EmployerUsername:  bounty.EmployerUsername,
		EmployerAccountId: bounty.EmployerAccountID,
		Title:             bounty.Title,
		Description:       bounty.Description,
		RewardAmount:      bounty.RewardAmount,
		Status:            string(bounty.Status),
		CreatedAt:         timestamppb.New(bounty.CreatedAt),
		UpdatedAt:         timestamppb.New(bounty.UpdatedAt),
	}
	if bounty.Deadline != nil {
		pbBounty.Deadline = timestamppb.New(*bounty.Deadline)
	}
	if bounty.SubmissionText != "" {
		pbBounty.SubmissionText = bounty.SubmissionText
	}
	// 填充当前 ACCEPTED 的猎人用户名
	for _, app := range bounty.Applications {
		if app.Status == models.AppStatusAccepted {
			pbBounty.HunterUsername = app.HunterUsername
			break
		}
	}
	return pbBounty
}

func convertBountyApplication(app *models.BountyApplication) *pb.BountyApplication {
	return &pb.BountyApplication{
		Id:               app.ID,
		BountyId:         app.BountyID,
		HunterUsername:   app.HunterUsername,
		HunterAccountId:  app.HunterAccountID,
		Status:           string(app.Status),
		CreatedAt:        timestamppb.New(app.CreatedAt),
		UpdatedAt:        timestamppb.New(app.UpdatedAt),
	}
}

func convertChatMessage(msg *models.ChatMessage, bountyID int64) *pb.ChatMessage {
	return &pb.ChatMessage{
		Id:             msg.ID,
		BountyId:       bountyID,
		SenderUsername: msg.SenderUsername,
		Content:        msg.Content,
		CreatedAt:      timestamppb.New(msg.CreatedAt),
	}
}

func convertPrivateConversation(conv *models.PrivateConversation, callerUsername string) *pb.PrivateConversation {
	return &pb.PrivateConversation{
		Id:            conv.ID,
		User1Username: conv.User1Username,
		User2Username: conv.User2Username,
		CreatedAt:     conv.CreatedAt.Unix(),
		UpdatedAt:     conv.UpdatedAt.Unix(),
	}
}

func convertPrivateMessage(msg *models.AllMessage) *pb.PrivateMessage {
	var convID int64
	if msg.ConversationID != nil {
		convID = *msg.ConversationID
	}
	return &pb.PrivateMessage{
		Id:             msg.ID,
		ConversationId: convID,
		SenderUsername: msg.SenderUsername,
		Content:        msg.Content,
		IsRead:         msg.IsRead,
		CreatedAt:      msg.CreatedAt.Unix(),
	}
}

func convertConversationSummary(conv *models.PrivateConversation, lastMsg *models.AllMessage, unreadCount int, callerUsername string) *pb.ConversationSummary {
	other := conv.User2Username
	if conv.User1Username == callerUsername {
		other = conv.User2Username
	} else {
		other = conv.User1Username
	}
	var lastContent string
	var lastAt int64
	if lastMsg != nil {
		lastContent = lastMsg.Content
		lastAt = lastMsg.CreatedAt.Unix()
	}
	return &pb.ConversationSummary{
		Id:               conv.ID,
		OtherUsername:    other,
		LastMessageContent: lastContent,
		LastMessageAt:    lastAt,
		UnreadCount:      int32(unreadCount),
	}
}

func convertComment(c *models.Comment, parentAuthorUsername string) *pb.Comment {
	pbComment := &pb.Comment{
		Id:                   c.ID,
		BountyId:             c.BountyID,
		ParentId:             0,
		AuthorUsername:        c.AuthorUsername,
		Content:              c.Content,
		CreatedAt:            c.CreatedAt.Unix(),
		ParentAuthorUsername: parentAuthorUsername,
	}
	if c.ParentID != nil {
		pbComment.ParentId = *c.ParentID
	}
	for _, reply := range c.Replies {
		converted := convertComment(&reply, c.AuthorUsername)
		pbComment.Replies = append(pbComment.Replies, converted)
	}
	return pbComment
}
