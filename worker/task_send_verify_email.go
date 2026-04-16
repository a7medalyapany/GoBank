package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	db "github.com/a7medalyapany/GoBank.git/db/sqlc"
	"github.com/a7medalyapany/GoBank.git/logger"
	"github.com/a7medalyapany/GoBank.git/mail"
	"github.com/a7medalyapany/GoBank.git/util"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// ─── Distribute (producer side)

func (distributor *RedisTaskDistributor) DistributeTaskSendVerifyEmail(
	ctx context.Context,
	payload *PayloadSendVerifyEmail,
	opts ...interface{},
) error {
	l := logger.G()

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal task payload: %w", err)
	}

	asynqOpts, err := normalizeAsynqOptions(opts...)
	if err != nil {
		return err
	}

	t := asynq.NewTask(TaskSendVerifyEmail, jsonPayload, asynqOpts...)

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

	verifyEmail, err := getOrCreateActiveVerifyEmail(ctx, processor.store, user)
	if err != nil {
		return err
	}

	// Build verification URL
	// verifyURL := fmt.Sprintf("%s/v1/verify_email?email_id=%d&secret_code=%s",
	verifyURL := fmt.Sprintf("%s/verify-email?email_id=%d&secret_code=%s", // redirects to frontend page, which then calls backend API to verify email
		processor.config.BASE_URL, verifyEmail.ID, verifyEmail.SecretCode)

	subject := "Welcome to GoBank — please verify your email"
	subject = verifyEmailSubject()
	content := fmt.Sprintf(`Hi %s,

	Thanks for registering. Click the link below to verify your email address:

	%s

	This link expires in 15 minutes. If you didn't create this account, ignore this email.
	`, user.FullName, verifyURL)

	err = processor.mailer.SendEmail(subject, content, []string{user.Email}, nil, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to send verification email: %w", err)
	}

	l.Info("processed task",
		zap.String("type", t.Type()),
		zap.ByteString("payload", t.Payload()),
		zap.String("email", user.Email),
		zap.Int64("verify_email_id", verifyEmail.ID),
	)

	return nil
}

type verifyEmailTaskHandler struct {
	store  *db.Store
	mailer mail.EmailSender
	config util.Config
}

func newVerifyEmailTaskHandler(store *db.Store, mailer mail.EmailSender, config util.Config) *verifyEmailTaskHandler {
	return &verifyEmailTaskHandler{
		store:  store,
		mailer: mailer,
		config: config,
	}
}

func (handler *verifyEmailTaskHandler) ProcessTaskSendVerifyEmail(ctx context.Context, payload *PayloadSendVerifyEmail) error {
	user, err := handler.store.GetUser(ctx, payload.Username)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	return handler.processTaskSendVerifyEmailForUser(ctx, user)
}

func (handler *verifyEmailTaskHandler) processTaskSendVerifyEmailForUser(ctx context.Context, user db.User) error {
	verifyEmail, err := getOrCreateActiveVerifyEmail(ctx, handler.store, user)
	if err != nil {
		return err
	}

	verifyURL := fmt.Sprintf(
		"%s/verify-email?email_id=%d&secret_code=%s",
		handler.config.BASE_URL,
		verifyEmail.ID,
		verifyEmail.SecretCode,
	)

	subject := "Welcome to GoBank â€” please verify your email"
	content := fmt.Sprintf(`Hi %s,

	Thanks for registering. Click the link below to verify your email address:

	%s

	This link expires in 15 minutes. If you didn't create this account, ignore this email.
	`, user.FullName, verifyURL)

	subject = verifyEmailSubject()
	if err := handler.mailer.SendEmail(subject, content, []string{user.Email}, nil, nil, nil); err != nil {
		return fmt.Errorf("failed to send verification email: %w", err)
	}

	logger.G().Info("sent verification email",
		zap.String("username", user.Username),
		zap.String("email", user.Email),
		zap.Int64("verify_email_id", verifyEmail.ID),
	)

	return nil
}

func normalizeAsynqOptions(opts ...interface{}) ([]asynq.Option, error) {
	asynqOpts := make([]asynq.Option, 0, len(opts))

	for _, opt := range opts {
		switch value := opt.(type) {
		case asynq.Option:
			asynqOpts = append(asynqOpts, value)
		case []asynq.Option:
			asynqOpts = append(asynqOpts, value...)
		default:
			return nil, fmt.Errorf("invalid task option type %T", opt)
		}
	}

	return asynqOpts, nil
}

func getOrCreateActiveVerifyEmail(ctx context.Context, store *db.Store, user db.User) (db.VerifyEmail, error) {
	verifyEmail, err := store.GetLatestActiveVerifyEmail(ctx, db.GetLatestActiveVerifyEmailParams{
		Username: user.Username,
		Email:    user.Email,
	})
	if err == nil {
		return verifyEmail, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return db.VerifyEmail{}, fmt.Errorf("failed to get active verify email: %w", err)
	}

	verifyEmail, err = store.CreateVerifyEmail(ctx, db.CreateVerifyEmailParams{
		Username:   user.Username,
		Email:      user.Email,
		SecretCode: util.RandomString(32),
	})
	if err != nil {
		return db.VerifyEmail{}, fmt.Errorf("failed to create verify email: %w", err)
	}

	return verifyEmail, nil
}

func verifyEmailSubject() string {
	return "Welcome to GoBank - please verify your email"
}
