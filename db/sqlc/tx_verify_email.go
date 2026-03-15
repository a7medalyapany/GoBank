package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

// VerifyEmailTxParams is input for verifying an email
type VerifyEmailTxParams struct {
	EmailId     int64
	SecretCode string
}

type VerifyEmailTxResult struct {
	User        User
	VerifyEmail VerifyEmail
}

func (store *Store) VerifyEmailTx(ctx context.Context, arg VerifyEmailTxParams) (VerifyEmailTxResult, error) {
 	var result VerifyEmailTxResult

 	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		result.VerifyEmail, err = q.UpdateVerifyEmail(ctx, UpdateVerifyEmailParams{
			arg.EmailId,
			arg.SecretCode,
		})
		if err != nil {
			return err
		}

		result.User, err = q.UpdateUser(ctx, UpdateUserParams{
			Username:        result.VerifyEmail.Username,
			IsEmailVerified: pgtype.Bool{Bool: true, Valid: true},
		})
		return err
 	})

	return result, err
}