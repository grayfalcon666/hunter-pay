package gapi

import (
	"context"
	"fmt"
	"time"

	db "simplebank/db/sqlc"
	"simplebank/mq"
	"simplebank/pb"
	"simplebank/util"
	val "simplebank/val"
	worker "simplebank/worker"

	"github.com/hibiken/asynq"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (server *Server) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	violations := validateCreateUserRequest(req)
	if violations != nil {
		return nil, invalidArgumentError(violations)
	}

	HashedPassword, err := util.HashPassword(req.GetPassword())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to hash password: %s", err)
	}

	// 生成唯一请求 ID 用于 Saga 幂等性
	requestId := uuid.New().String()

	arg := db.CreateUserTxParams{
		CreateUserParams: db.CreateUserParams{
			Username:       req.GetUsername(),
			HashedPassword: HashedPassword,
			FullName:       req.GetFullName(),
			Email:          req.GetEmail(),
		},
		AfterCreate: func(user db.User) error {
			// 1. 发送验证邮件
			payload := &worker.PayloadSendVerifyEmail{Username: user.Username}
			opts := []asynq.Option{
				asynq.MaxRetry(10),
				asynq.ProcessIn(10 * time.Second),
				asynq.Queue("critical"),
			}
			err = server.taskDistributor.DistributeTaskSendVerifyEmail(ctx, payload, opts...)
			if err != nil {
				return status.Error(codes.Internal, "failed to distribute task to send verify email")
			}

			// 2. 发布用户创建事件（异步，不阻塞）
			if server.userProducer != nil {
				event := &mq.UserCreatedEvent{
					Username:  user.Username,
					Email:     user.Email,
					FullName:  user.FullName,
					RequestId: requestId,
					CreatedAt: time.Now(),
				}
				// 后台发布事件，不阻塞响应
				go func() {
					ctx := context.Background()
					if err := server.userProducer.PublishUserCreatedEvent(ctx, event); err != nil {
						// 记录错误但不影响用户注册
						// 在真实场景中，应该发送到日志系统或监控
						fmt.Printf("发布 UserCreatedEvent 失败：%v\n", err)
					}
				}()
			}

			return nil
		},
	}

	result, err := server.store.CreateUserTx(ctx, arg)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				return nil, status.Error(codes.AlreadyExists, err.Error())
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to create user: %s", err)
	}

	rsp := &pb.CreateUserResponse{
		User: convertUser(result.User),
	}
	return rsp, nil
}

func validateCreateUserRequest(req *pb.CreateUserRequest) (violations []*errdetails.BadRequest_FieldViolation) {
	if err := val.ValidateUsername(req.GetUsername()); err != nil {
		violations = append(violations, fieldViolation("username", err))
	}

	if err := val.ValidateEmail(req.GetEmail()); err != nil {
		violations = append(violations, fieldViolation("email", err))
	}

	if err := val.ValidatePassword(req.GetPassword()); err != nil {
		violations = append(violations, fieldViolation("password", err))
	}

	return violations
}
