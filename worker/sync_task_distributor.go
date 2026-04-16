package worker

import (
	"context"
	"errors"
	"fmt"

	db "github.com/a7medalyapany/GoBank.git/db/sqlc"
	"github.com/a7medalyapany/GoBank.git/mail"
	"github.com/a7medalyapany/GoBank.git/util"
	"github.com/jackc/pgx/v5"
)

type SyncTaskDistributor struct {
	handler *verifyEmailTaskHandler
}

type syncVerifyEmailTaskDistributor interface {
	DistributeTaskSendVerifyEmailForUser(ctx context.Context, user db.User) error
}

func NewSyncTaskDistributor(store *db.Store, mailer mail.EmailSender, config util.Config) TaskDistributor {
	return &SyncTaskDistributor{
		handler: newVerifyEmailTaskHandler(store, mailer, config),
	}
}

func (distributor *SyncTaskDistributor) DistributeTaskSendVerifyEmail(
	ctx context.Context,
	payload *PayloadSendVerifyEmail,
	_ ...interface{},
) error {
	if err := distributor.handler.ProcessTaskSendVerifyEmail(ctx, payload); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("user does not exist: %w", err)
		}
		return err
	}

	return nil
}

func (distributor *SyncTaskDistributor) DistributeTaskSendVerifyEmailForUser(ctx context.Context, user db.User) error {
	return distributor.handler.processTaskSendVerifyEmailForUser(ctx, user)
}
