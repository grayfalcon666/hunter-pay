package gapi

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/grayfalcon666/payment-service/models"
	"github.com/grayfalcon666/payment-service/mq"
	"github.com/grayfalcon666/payment-service/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (server *Server) CreateWithdrawal(ctx context.Context, req *pb.CreateWithdrawalRequest) (*pb.CreateWithdrawalResponse, error) {
	if req.GetAmount() <= 0 || req.GetAccountId() <= 0 || req.GetAlipayAccount() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "金额、账户 ID 或支付宝账号不合法")
	}

	authPayload, err := server.verifyUserAndAccount(ctx, req.GetAccountId())
	if err != nil {
		return nil, err
	}

	outBizNo := uuid.New().String()
	idempotencyKey := fmt.Sprintf("withdraw_freeze_%s", outBizNo)

	withdrawal := &models.Withdrawal{
		Username:       authPayload.Username,
		AccountID:      req.GetAccountId(),
		Amount:         req.GetAmount(),
		AlipayAccount:  req.GetAlipayAccount(),
		AlipayRealName: req.GetAlipayRealName(),
		OutBizNo:       outBizNo,
		Status:         "INIT",
	}

	// Step 1: 先写本地 DB（占位）
	if err := server.store.CreateWithdrawal(ctx, withdrawal); err != nil {
		return nil, status.Errorf(codes.Internal, "创建提现记录失败: %v", err)
	}

	// Step 2: 执行 SimpleBank Freeze（冻结用户可用余额）
	// Freeze 成功后，用户的可用余额减少，冻结余额增加
	err = server.bankClient.Freeze(ctx, req.GetAccountId(), req.GetAmount(), idempotencyKey)
	if err != nil {
		// Freeze 失败：标记为失败
		log.Printf("提现 Freeze 失败，outBizNo=%s, err=%v\n", outBizNo, err)
		server.store.UpdateWithdrawalStatus(context.Background(), outBizNo, "FAILED", fmt.Sprintf("冻结余额失败: %v", err))
		return nil, status.Errorf(codes.FailedPrecondition, "冻结用户余额失败: %v", err)
	}

	// Step 3: 推进状态 INIT → PROCESSING（乐观锁，幂等保证）
	rowsAffected, err := server.store.TryUpdateWithdrawalStatusToProcessing(ctx, outBizNo)
	if err != nil {
		log.Printf("更新提现状态失败，outBizNo=%s, err=%v\n", outBizNo, err)
		// 乐观锁失败说明已经处理过了，按幂等成功处理
		if rowsAffected == 0 {
			return &pb.CreateWithdrawalResponse{
				Message:  "提现申请已提交（幂等重复）",
				OutBizNo: outBizNo,
			}, nil
		}
		return nil, status.Errorf(codes.Internal, "更新提现状态失败: %v", err)
	}

	// Step 4: 发送 MQ 消息
	msg := &mq.WithdrawalMessage{
		WithdrawalID:   withdrawal.ID,
		Username:       authPayload.Username,
		AccountID:      req.GetAccountId(),
		Amount:         req.GetAmount(),
		AlipayAccount:  req.GetAlipayAccount(),
		AlipayRealName: req.GetAlipayRealName(),
		OutBizNo:       outBizNo,
	}
	err = server.mqProducer.PublishWithdrawalProcess(msg)
	if err != nil {
		// MQ 发送失败，执行 Saga 补偿：用 Unfreeze 回滚冻结
		log.Printf("提现 MQ 发送失败，执行 Unfreeze 回滚，outBizNo=%s, err=%v\n", outBizNo, err)
		rollbackKey := fmt.Sprintf("withdraw_rollback_%s", outBizNo)
		rollbackErr := server.bankClient.Unfreeze(context.Background(), req.GetAccountId(), req.GetAmount(), rollbackKey)
		if rollbackErr != nil {
			log.Printf("严重错误！提现 MQ 发送失败且 Unfreeze 回滚也失败！outBizNo=%s, rollbackErr=%v\n", outBizNo, rollbackErr)
			server.store.UpdateWithdrawalStatus(context.Background(), outBizNo, "FAILED", "MQ发送失败且Unfreeze回滚失败，请人工处理")
		} else {
			log.Printf("提现 MQ 发送失败已 Unfreeze 回滚，outBizNo=%s\n", outBizNo)
			server.store.UpdateWithdrawalStatus(context.Background(), outBizNo, "FAILED", "MQ发送失败已回滚")
		}
		return nil, status.Errorf(codes.Internal, "提现提交失败，请重试: %v", err)
	}

	return &pb.CreateWithdrawalResponse{
		Message:  "提现申请已提交，系统正在处理中",
		OutBizNo: outBizNo,
	}, nil
}
