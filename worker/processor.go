package worker

import (
	"context"

	db "github.com/a7medalyapany/GoBank.git/db/sqlc"
	"github.com/a7medalyapany/GoBank.git/logger"
	"github.com/a7medalyapany/GoBank.git/mail"
	"github.com/a7medalyapany/GoBank.git/util"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

// TaskProcessor is the consumer-side interface.
// Add a new method signature here for every new task type.
type TaskProcessor interface {
	Start() error
	ProcessTaskSendVerifyEmail(ctx context.Context, t *asynq.Task) error
}

type RedisTaskProcessor struct {
	server *asynq.Server
	store  *db.Store
	mailer mail.EmailSender
	config util.Config
}

func NewRedisTaskProcessor(redisOpt asynq.RedisClientOpt, store *db.Store, mailer mail.EmailSender, config util.Config) TaskProcessor {
	l := logger.G()
	server := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				QueueCritical: 10,
				QueueDefault:  5,
				QueueLow:      1,
			},
			ErrorHandler: asynq.ErrorHandlerFunc(
				func(ctx context.Context, task *asynq.Task, err error) {
					l.Error("failed to process task:", zap.String("type", task.Type()), zap.String("payload", string(task.Payload())), zap.Error(err))
				},
			),
			Logger: NewLogger(),
		},
	)

	return &RedisTaskProcessor{
		server: server,
		store:  store,
		mailer: mailer,
		config: config,
	}
}

// Start registers all task handlers and begins polling Redis.
// Add one mux.HandleFunc line here for every new task type.
func (processor *RedisTaskProcessor) Start() error {
	mux := asynq.NewServeMux()

	mux.HandleFunc(TaskSendVerifyEmail, processor.ProcessTaskSendVerifyEmail)

	return processor.server.Start(mux)
}
