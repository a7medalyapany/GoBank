package worker

import (
	"context"

	db "github.com/a7medalyapany/GoBank.git/db/sqlc"
	"github.com/hibiken/asynq"
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
}

func NewRedisTaskProcessor(redisOpt asynq.RedisClientOpt, store *db.Store) TaskProcessor {
	server := asynq.NewServer(
		redisOpt,
		asynq.Config{},
	)

	return &RedisTaskProcessor{
		server: server,
		store:  store,
	}
}

// Start registers all task handlers and begins polling Redis.
// Add one mux.HandleFunc line here for every new task type.
func (processor *RedisTaskProcessor) Start() error {
	mux := asynq.NewServeMux()

	mux.HandleFunc(TaskSendVerifyEmail, processor.ProcessTaskSendVerifyEmail)

	return processor.server.Start(mux)
}