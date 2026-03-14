package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/a7medalyapany/GoBank.git/logger"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// ─── Distribute (producer side)

func (distributor *RedisTaskDistributor) DistributeTaskSendVerifyEmail(
	ctx context.Context,
	payload *PayloadSendVerifyEmail,
	opts ...asynq.Option,
) error {
	l := logger.G()

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal task payload: %w", err)
	}

	t := asynq.NewTask(TaskSendVerifyEmail, jsonPayload, opts...)

	info, err := distributor.client.EnqueueContext(ctx, t)
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	l.Info("enqueued task",
		zap.String("type", t.Type()),
		zap.ByteString("payload", t.Payload()),
		zap.String("queue", info.Queue),
		zap.Int("max_retry", info.MaxRetry),
	)

	return nil
}

// ─── Process (consumer side)

func (processor *RedisTaskProcessor) ProcessTaskSendVerifyEmail(ctx context.Context, t *asynq.Task) error {
	l := logger.G()

	var payload PayloadSendVerifyEmail
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	user, err := processor.store.GetUser(ctx, payload.Username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// User was deleted after the task was enqueued — no point retrying.
			return fmt.Errorf("user does not exist: %w", asynq.SkipRetry)
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	// TODO: send verification email to user.Email

	l.Info("processed task",
		zap.String("type", t.Type()),
		zap.ByteString("payload", t.Payload()),
		zap.String("email", user.Email),
	)

	return nil
}