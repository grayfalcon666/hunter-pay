package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	db "simplebank/db/sqlc"
	"simplebank/util"

	"github.com/hibiken/asynq"
)

func (processor *RedisTaskProcessor) ProcessTaskSendVerifyEmail(ctx context.Context, task *asynq.Task) error {
	var payload PayloadSendVerifyEmail
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		slog.ErrorContext(
			ctx,
			"failed to unmarshal verify email task payload",
			"error", err, // 错误详情
			"task_type", task.Type(),
			"task_id", task.ResultWriter().TaskID(),
		)
		return fmt.Errorf("failed to unmarshal payload: %w", asynq.SkipRetry)
	}

	user, err := processor.store.GetUser(ctx, payload.Username)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("user doesn't exist: %w", asynq.SkipRetry)
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	verifyEmail, err := processor.store.CreateVerifyEmail(ctx, db.CreateVerifyEmailParams{
		Username:   user.Username,
		Email:      user.Email,
		SecretCode: util.RandomString(32),
	})
	if err != nil {
		return fmt.Errorf("failed to create verify email: %w", err)
	}

	// verifyUrl := fmt.Sprintf("https://api.simplebank.website:4443/v1/verify_email?email_id=%d&secret_code=%s",
	// 	verifyEmail.ID, verifyEmail.SecretCode)
	verifyUrl := fmt.Sprintf("http://192.168.0.104:11452/v1/verify_email?email_id=%d&secret_code=%s",
		verifyEmail.ID, verifyEmail.SecretCode)

	content := fmt.Sprintf(`Hello %s,<br/>
    Thank you for registering with us!<br/>
    Please <a href="%s">click here</a> to verify your email address.<br/>`,
		user.FullName, verifyUrl)

	err = processor.mailer.SendEmail("Verify your email", content, []string{user.Email}, nil, nil, nil)
	logger := slog.With(
		slog.String("username", user.Username),
		slog.String("email", user.Email),
	)

	if err != nil {
		logger.Error("failed to send verify email", slog.String("error", err.Error()))
		return fmt.Errorf("failed to send verify email: %w", err)
	}

	logger.Info("success to send verify email")
	return nil
}
