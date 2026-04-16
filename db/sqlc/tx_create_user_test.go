package db

import (
	"context"
	"errors"
	"testing"

	"github.com/a7medalyapany/GoBank.git/util"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func randomCreateUserTxParams(t *testing.T) CreateUserTxParams {
	t.Helper()

	hashedPassword, err := util.HashPassword(util.RandomString(12))
	require.NoError(t, err)

	return CreateUserTxParams{
		CreateUserParams: CreateUserParams{
			Username:       util.RandomOwner(),
			HashedPassword: hashedPassword,
			FullName:       util.RandomOwner(),
			Email:          util.RandomEmail(),
		},
	}
}

func TestCreateUserTxWithNilAfterCreate(t *testing.T) {
	store := NewStore(testDB)
	ctx := context.Background()
	arg := randomCreateUserTxParams(t)

	result, err := store.CreateUserTx(ctx, arg)
	require.NoError(t, err)
	require.Equal(t, arg.Username, result.User.Username)

	storedUser, err := store.GetUser(ctx, arg.Username)
	require.NoError(t, err)
	require.Equal(t, arg.Email, storedUser.Email)
}

func TestCreateUserTxCallsAfterCreate(t *testing.T) {
	store := NewStore(testDB)
	ctx := context.Background()
	arg := randomCreateUserTxParams(t)

	called := false
	arg.AfterCreate = func(user User) error {
		called = true
		require.Equal(t, arg.Username, user.Username)
		require.Equal(t, arg.Email, user.Email)
		return nil
	}

	result, err := store.CreateUserTx(ctx, arg)
	require.NoError(t, err)
	require.Equal(t, arg.Username, result.User.Username)
	require.True(t, called)
}

func TestCreateUserTxRollsBackOnAfterCreateError(t *testing.T) {
	store := NewStore(testDB)
	ctx := context.Background()
	arg := randomCreateUserTxParams(t)

	afterCreateErr := errors.New("after create failed")
	arg.AfterCreate = func(user User) error {
		require.Equal(t, arg.Username, user.Username)
		return afterCreateErr
	}

	_, err := store.CreateUserTx(ctx, arg)
	require.Error(t, err)
	require.ErrorIs(t, err, afterCreateErr)

	_, err = store.GetUser(ctx, arg.Username)
	require.Error(t, err)
	require.ErrorIs(t, err, pgx.ErrNoRows)
}
