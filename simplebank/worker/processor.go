package worker

import (
	"context"
	"log/slog"
	db "simplebank/db/sqlc"
	"simplebank/mail"

	"github.com/hibiken/asynq"
)

type TaskProcessor interface {
	Start() error
	Shutdown()
	ProcessTaskSendVerifyEmail(ctx context.Context, task *asynq.Task) error
}

type RedisTaskProcessor struct {
	server *asynq.Server
	store  db.Store
	mailer mail.EmailSender
}

func NewRedisTaskProcessor(redisOpt asynq.RedisClientOpt, store db.Store, mailer mail.EmailSender) TaskProcessor {
	server := asynq.NewServer(redisOpt,
		asynq.Config{
			Queues: map[string]int{
				"critical": 10,
				"default":  5,
			},
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				slog.ErrorContext(
					ctx,
					"process task failed",
					"type", task.Type(),
					"payload", task.Payload(),
				)
			}),
			Logger: NewLogger(),
		})

	return &RedisTaskProcessor{
		server: server,
		store:  store,
		mailer: mailer,
	}
}

func (processor *RedisTaskProcessor) Start() error {
	mux := asynq.NewServeMux()

	// 注册：当看到 TaskSendVerifyEmail 任务时，执行对应的 Process 函数
	mux.HandleFunc(TaskSendVerifyEmail, processor.ProcessTaskSendVerifyEmail)

	return processor.server.Run(mux)
}

func (processor *RedisTaskProcessor) Shutdown() {
	processor.server.Shutdown()
}
