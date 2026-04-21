package gapi

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/grayfalcon666/escrow-bounty/models"
	"github.com/grayfalcon666/escrow-bounty/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func (server *Server) ListComments(ctx context.Context, req *pb.ListCommentsRequest) (*pb.ListCommentsResponse, error) {
	_, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}

	bountyID := req.GetBountyId()
	if bountyID <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "无效的悬赏 ID")
	}

	comments, err := server.store.ListComments(ctx, bountyID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "查询评论失败: %v", err)
	}

	// 收集所有评论作者，去重
	usernameSet := make(map[string]struct{})
	for _, c := range comments {
		usernameSet[c.AuthorUsername] = struct{}{}
	}
	usernames := make([]string, 0, len(usernameSet))
	for u := range usernameSet {
		usernames = append(usernames, u)
	}

	// 批量查头像
	avatarMap, _ := server.store.GetAvatarUrlsByUsernames(ctx, usernames)

	// 建立 id -> comment 的映射，用于查找 reply_to 的内容和作者
	idToComment := make(map[int64]*models.Comment)
	for i := range comments {
		idToComment[comments[i].ID] = &comments[i]
	}

	// 返回该 bounty 的所有评论（扁平列表）
	var pbComments []*pb.Comment
	for _, c := range comments {
		replyToAuthor := ""
		replyToUsername := ""
		replyToContent := ""
		if c.ReplyToID != nil {
			if ref := idToComment[*c.ReplyToID]; ref != nil {
				replyToAuthor = ref.AuthorUsername
				replyToUsername = ref.AuthorUsername
				replyToContent = ref.Content
			}
		}
		pbComment := convertComment(ctx, server, &c, replyToAuthor)
		pbComment.AuthorAvatarUrl = avatarMap[c.AuthorUsername]
		pbComment.ReplyToUsername = replyToUsername
		pbComment.ReplyToContent = replyToContent
		pbComments = append(pbComments, pbComment)
	}

	return &pb.ListCommentsResponse{
		Comments: pbComments,
	}, nil
}

func (server *Server) CreateComment(ctx context.Context, req *pb.CreateCommentRequest) (*pb.CreateCommentResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}
	author := authPayload.Username

	bountyID := req.GetBountyId()
	if bountyID <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "无效的悬赏 ID")
	}

	content := strings.TrimSpace(req.GetContent())
	if content == "" {
		return nil, status.Errorf(codes.InvalidArgument, "评论内容不能为空")
	}
	if int64(len(content)) > 2000 {
		return nil, status.Errorf(codes.InvalidArgument, "评论内容不能超过 2000 字符")
	}

	// reply_to_id: 被回复的评论ID，0 表示发布根评论
	var replyToID *int64
	if req.GetReplyToId() > 0 {
		replyToID = &req.ReplyToId
	}

	// image_id: 评论图片
	var imageID *int64
	if req.GetImageId() > 0 {
		imageID = &req.ImageId
	}

	comment, err := server.store.CreateComment(ctx, bountyID, replyToID, author, content, imageID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "悬赏或回复的评论不存在")
		}
		if strings.Contains(err.Error(), "does not belong to this bounty") {
			return nil, status.Errorf(codes.InvalidArgument, "回复的评论不属于此悬赏")
		}
		return nil, status.Errorf(codes.Internal, "创建评论失败: %v", err)
	}

	// 填充被回复评论的作者名
	replyToAuthor := ""
	replyToUsername := ""
	replyToContent := ""
	if replyToID != nil {
		replyTo, err := server.store.GetComment(ctx, *replyToID)
		if err == nil {
			replyToAuthor = replyTo.AuthorUsername
			replyToUsername = replyTo.AuthorUsername
			replyToContent = replyTo.Content
		}
	}

	pbComment := convertComment(ctx, server, comment, replyToAuthor)
	pbComment.ReplyToUsername = replyToUsername
	pbComment.ReplyToContent = replyToContent

	return &pb.CreateCommentResponse{
		Comment: pbComment,
	}, nil
}

func (server *Server) DeleteComment(ctx context.Context, req *pb.DeleteCommentRequest) (*pb.DeleteCommentResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}
	caller := authPayload.Username

	commentID := req.GetCommentId()
	if commentID <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "无效的评论 ID")
	}

	// DB事务删除评论+子评论+图片记录，返回被删图片路径
	deletedPaths, err := server.store.DeleteCommentCascade(ctx, commentID, caller)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "评论不存在")
		}
		if strings.Contains(err.Error(), "permission denied") {
			return nil, status.Errorf(codes.PermissionDenied, "只能删除自己的评论")
		}
		return nil, status.Errorf(codes.Internal, "删除评论失败: %v", err)
	}

	// 业务层：清理物理图片文件
	for _, path := range deletedPaths {
		fullPath := filepath.Join(uploadBasePath, path)
		os.Remove(fullPath)
	}

	return &pb.DeleteCommentResponse{}, nil
}

func (server *Server) ListUserComments(ctx context.Context, req *pb.ListUserCommentsRequest) (*pb.ListUserCommentsResponse, error) {
	_, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}

	username := req.GetUsername()
	if username == "" {
		return nil, status.Errorf(codes.InvalidArgument, "username is required")
	}

	limit := int(req.GetPageSize())
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := int((req.GetPageId() - 1) * req.GetPageSize())
	if offset < 0 {
		offset = 0
	}

	comments, err := server.store.ListCommentsByUsername(ctx, username, limit, offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "查询用户评论失败: %v", err)
	}

	// 收集所有评论作者，去重查头像
	usernameSet := make(map[string]struct{})
	for _, c := range comments {
		usernameSet[c.AuthorUsername] = struct{}{}
	}
	usernames := make([]string, 0, len(usernameSet))
	for u := range usernameSet {
		usernames = append(usernames, u)
	}
	avatarMap, _ := server.store.GetAvatarUrlsByUsernames(ctx, usernames)

	// 建立 id -> comment 映射，查找 replyToUsername 和 replyToContent
	idToComment := make(map[int64]*models.Comment)
	for i := range comments {
		idToComment[comments[i].ID] = &comments[i]
	}

	var pbComments []*pb.Comment
	for _, c := range comments {
		replyToAuthor := ""
		replyToUsername := ""
		replyToContent := ""
		if c.ReplyToID != nil {
			if replyTo, err := server.store.GetComment(ctx, *c.ReplyToID); err == nil {
				replyToAuthor = replyTo.AuthorUsername
			}
			if ref := idToComment[*c.ReplyToID]; ref != nil {
				replyToUsername = ref.AuthorUsername
				replyToContent = ref.Content
			}
		}
		pbComment := convertComment(ctx, server, &c, replyToAuthor)
		pbComment.AuthorAvatarUrl = avatarMap[c.AuthorUsername]
		pbComment.ReplyToUsername = replyToUsername
		pbComment.ReplyToContent = replyToContent
		pbComments = append(pbComments, pbComment)
	}

	return &pb.ListUserCommentsResponse{
		Comments: pbComments,
	}, nil
}
