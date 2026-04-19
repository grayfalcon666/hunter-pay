package gapi

import (
	"context"
	"log"
	"time"

	"github.com/grayfalcon666/escrow-bounty/models"
	"github.com/grayfalcon666/escrow-bounty/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (server *Server) CreateBounty(ctx context.Context, req *pb.CreateBountyRequest) (*pb.CreateBountyResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, err
	}

	if req.GetRewardAmount() <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "悬赏金额必须大于 0")
	}

	employerUsername := req.GetEmployerUsername()
	if employerUsername == "" {
		employerUsername = authPayload.Username
	}

	employerAccountID := req.GetEmployerAccountId()

	// 如果前端没有提供 employer_account_id，通过 gRPC 查询用户账户列表找到第一个账户
	if employerAccountID <= 0 {
		if server.bankClient == nil {
			return nil, status.Errorf(codes.FailedPrecondition, "无法获取账户信息，请先创建账户")
		}
		accounts, err := server.bankClient.ListAccounts(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "查询用户账户失败: %v", err)
		}
		if len(accounts) == 0 {
			return nil, status.Errorf(codes.FailedPrecondition, "您还没有创建账户，请先创建账户")
		}
		employerAccountID = accounts[0].Id
		log.Printf("自动找到雇主 %s 的账户 ID: %d", employerUsername, employerAccountID)
	}

	bounty := &models.Bounty{
		EmployerUsername:  employerUsername,
		EmployerAccountID: employerAccountID,
		Title:             req.GetTitle(),
		Description:       req.GetDescription(),
		RewardAmount:      req.GetRewardAmount(),
	}

	if deadlineTs := req.GetDeadlineTimestamp(); deadlineTs > 0 {
		deadline := time.Unix(deadlineTs, 0)
		bounty.Deadline = &deadline
	}

	// 调用带有 Saga 分布式事务逻辑的 Store 方法（内部冻结，无需平台账户）
	err = server.store.PublishBounty(ctx, bounty, server.bankClient, employerAccountID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "发布悬赏失败: %v", err)
	}

	return &pb.CreateBountyResponse{
		Bounty: convertBounty(bounty),
	}, nil
}
