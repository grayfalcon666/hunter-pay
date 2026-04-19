package gapi

import (
	"context"
	"errors"
	"strings"

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

	// Build tree: only top-level comments (parent_id is null)
	var topLevel []*pb.Comment
	for _, c := range comments {
		if c.ParentID == nil {
			topLevel = append(topLevel, convertComment(&c, ""))
		}
	}

	return &pb.ListCommentsResponse{
		Comments: topLevel,
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

	var parentID *int64
	if req.GetParentId() > 0 {
		parentID = &req.ParentId
	}

	comment, err := server.store.CreateComment(ctx, bountyID, parentID, author, content)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "悬赏或回复的评论不存在")
		}
		if strings.Contains(err.Error(), "does not belong to this bounty") {
			return nil, status.Errorf(codes.InvalidArgument, "回复的评论不属于此悬赏")
		}
		return nil, status.Errorf(codes.Internal, "创建评论失败: %v", err)
	}

	return &pb.CreateCommentResponse{
		Comment: convertComment(comment, ""),
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

	if err := server.store.DeleteComment(ctx, commentID, caller); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "评论不存在")
		}
		if strings.Contains(err.Error(), "permission denied") {
			return nil, status.Errorf(codes.PermissionDenied, "只能删除自己的评论")
		}
		return nil, status.Errorf(codes.Internal, "删除评论失败: %v", err)
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

	var pbComments []*pb.Comment
	for _, c := range comments {
		pbComments = append(pbComments, convertComment(&c, ""))
	}

	return &pb.ListUserCommentsResponse{
		Comments: pbComments,
	}, nil
}
