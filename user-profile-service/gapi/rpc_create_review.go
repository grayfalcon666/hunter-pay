package gapi

import (
	"context"
	"log"

	"github.com/grayfalcon666/user-profile-service/mq"
	"github.com/grayfalcon666/user-profile-service/models"
	"github.com/grayfalcon666/user-profile-service/pb"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (server *Server) CreateReview(ctx context.Context, req *pb.CreateReviewRequest) (*pb.CreateReviewResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}

	reviewerUsername := authPayload.Username

	// 评分校验
	if req.Rating < 1 || req.Rating > 5 {
		return nil, status.Errorf(codes.InvalidArgument, "评分必须在 1~5 之间")
	}

	// 不能评价自己
	if reviewerUsername == req.ReviewedUsername {
		return nil, status.Errorf(codes.InvalidArgument, "不能评价自己")
	}

	// reviewedUsername 不能为空
	if req.ReviewedUsername == "" {
		return nil, status.Errorf(codes.InvalidArgument, "被评价人不能为空")
	}

	// 检查是否已评价过该悬赏（每人只能评价一次）
	existingReviews, err := server.store.GetReviewsByUser(ctx, req.ReviewedUsername)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "查询评价失败: %v", err)
	}
	for _, r := range existingReviews {
		if r.ReviewerUsername == reviewerUsername && r.BountyID == req.BountyId {
			return nil, status.Errorf(codes.AlreadyExists, "您已评价过该悬赏")
		}
	}

	params := &models.CreateReviewParams{
		ReviewedUsername: req.ReviewedUsername,
		BountyID:         req.BountyId,
		Rating:           int(req.Rating),
		Comment:          req.Comment,
		ReviewType:       models.ReviewType(req.ReviewType),
	}

	review, err := server.store.CreateReview(ctx, reviewerUsername, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "创建评价失败: %v", err)
	}

	// 更新 task_record 的评分并触发二次履约重算
	go server.handleReviewFulfillmentUpdate(review)

	return &pb.CreateReviewResponse{
		Review: convertReview(review),
	}, nil
}

// handleReviewFulfillmentUpdate 处理评价带来的履约重算
func (server *Server) handleReviewFulfillmentUpdate(review *models.UserReview) {
	ctx := context.Background()

	// 根据 ReviewType 确定被更新的 task_record 的用户名和角色
	// EMPLOYER_TO_HUNTER → 猎人被评价，更新猎人的 task_record（role=HUNTER），更新 employer_rating
	// HUNTER_TO_EMPLOYER → 雇主被评价，更新雇主的 task_record（role=EMPLOYER），更新 hunter_rating
	var taskUsername, role string
	if review.ReviewType == models.ReviewEmployerToHunter {
		taskUsername = review.ReviewedUsername // 猎人
		role = string(models.RoleHunter)
	} else {
		taskUsername = review.ReviewedUsername // 雇主
		role = string(models.RoleEmployer)
	}

	// 更新 task_record 的评分（UpdateTaskRecordRating 内部判断是否设置 finalized=true）
	if err := server.store.UpdateTaskRecordRating(ctx, review.BountyID, taskUsername, review.ReviewType, review.Rating); err != nil {
		log.Printf("handleReviewFulfillmentUpdate: 更新评分失败 (bounty=%d, taskUsername=%s): %v",
			review.BountyID, taskUsername, err)
		return
	}

	// 查询更新后的记录，确认 rating_finalized 是否为 true（双方都评价了才触发 MQ 重算）
	updatedRecord, err := server.store.GetTaskRecordByBountyAndUsername(ctx, review.BountyID, taskUsername)
	if err != nil {
		log.Printf("handleReviewFulfillmentUpdate: 查询更新后记录失败: %v", err)
		return
	}

	// 触发履约指数重算（只有双方都评价了才触发）
	if updatedRecord.RatingFinalized && server.eventProducer != nil {
		// 发布被评价人的重算事件
		reviewedEvent := &mq.FulfillmentRecalcEvent{
			Username:  taskUsername,
			Role:      role,
			BountyID:  review.BountyID,
			RequestID: uuid.New().String(),
		}
		if err := server.eventProducer.PublishFulfillmentRecalcEvent(ctx, reviewedEvent); err != nil {
			log.Printf("handleReviewFulfillmentUpdate: 发布 MQ 事件失败 (reviewed): %v", err)
		} else {
			log.Printf("handleReviewFulfillmentUpdate: 履约重算事件已发布 (username=%s, role=%s, bounty_id=%d)",
				taskUsername, role, review.BountyID)
		}

		// 发布评价人的重算事件（对端角色）
		reviewerRole := string(models.RoleEmployer)
		if role == string(models.RoleEmployer) {
			reviewerRole = string(models.RoleHunter)
		}
		reviewerEvent := &mq.FulfillmentRecalcEvent{
			Username:  review.ReviewerUsername,
			Role:      reviewerRole,
			BountyID:  review.BountyID,
			RequestID: uuid.New().String(),
		}
		if err := server.eventProducer.PublishFulfillmentRecalcEvent(ctx, reviewerEvent); err != nil {
			log.Printf("handleReviewFulfillmentUpdate: 发布 MQ 事件失败 (reviewer): %v", err)
		} else {
			log.Printf("handleReviewFulfillmentUpdate: 履约重算事件已发布 (username=%s, role=%s, bounty_id=%d)",
				review.ReviewerUsername, reviewerRole, review.BountyID)
		}
	}
}
