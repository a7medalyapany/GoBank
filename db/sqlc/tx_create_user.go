package db

import "context"

// CreateUserTxParams is input for creating a user
type CreateUserTxParams struct {
	CreateUserParams
	AfterCreate func(user User) error
}

type CreateUserTxResult struct {
	User User
}

// CreateUserTx performs a database transaction for creating a user and optionally executing the AfterCreate callback.
func (store *Store) CreateUserTx(ctx context.Context, arg CreateUserTxParams) (CreateUserTxResult, error) {
	var result CreateUserTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		result.User, err = q.CreateUser(ctx, arg.CreateUserParams)
		if err != nil {
			return err
		}

		if arg.AfterCreate != nil {
			return arg.AfterCreate(result.User)
		}

		return nil
	})

	return result, err
}
