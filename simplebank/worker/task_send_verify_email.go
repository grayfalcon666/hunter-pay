package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
)

type PayloadSendVerifyEmail struct {
	Username string `json:"username"`
}

const TaskSendVerifyEmail = "task:send_verify_email"

func (distributor *RedisTaskDistributor) DistributeTaskSendVerifyEmail(
	ctx context.Context,
	payload *PayloadSendVerifyEmail,
	opts ...asynq.Option,
) error {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal task payload: %w", err)
	}

	task := asynq.NewTask(TaskSendVerifyEmail, jsonPayload, opts...)
	info, err := distributor.client.EnqueueContext(ctx, task)

	logger := slog.With(
		slog.String("task_id", info.ID),
		slog.String("queue", info.Queue),
		slog.String("type", task.Type()),
		slog.String("payload", string(task.Payload())),
	)

	if err != nil {
		logger.Error("failed to enqueue task", slog.String("error", err.Error()))
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	logger.Info("enqueued a task")

	return nil
}
