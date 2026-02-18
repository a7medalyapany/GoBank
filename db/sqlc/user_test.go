package db

import (
	"context"
	"testing"
	"time"

	"github.com/a7medalyapany/GoBank.git/util"
	"github.com/stretchr/testify/require"
)

func createRandomUser(t *testing.T) User {
	arg := CreateUserParams{
		Username: util.RandomOwner(),
		HashedPassword: "password",
		FullName: util.RandomOwner(),
		Email:    util.RandomEmail(),
	}

	user, err := testQueries.CreateUser(context.Background(), arg)

	require.NoError(t, err)
	require.NotEmpty(t, user)

	require.Equal(t, arg.Username, user.Username)
	require.Equal(t, arg.FullName, user.FullName)
	require.Equal(t, arg.Email, user.Email)
	require.Equal(t, arg.HashedPassword, user.HashedPassword)

	require.True(t, user.PasswordChangedAt.Time.IsZero())
	require.NotZero(t, user.CreatedAt)
	return user
}

func TestCreateUser(t *testing.T) {
	createRandomUser(t)
}

func TestGetUser(t *testing.T) {
	user := createRandomUser(t)

	retrievedUser, err := testQueries.GetUser(context.Background(), user.Username)
	require.NoError(t, err)
	require.NotEmpty(t, retrievedUser)

	require.Equal(t, user.Username, retrievedUser.Username)
	require.Equal(t, user.HashedPassword, retrievedUser.HashedPassword)
	require.Equal(t, user.FullName, retrievedUser.FullName)
	require.Equal(t, user.Email, retrievedUser.Email)
	require.WithinDuration(t, user.PasswordChangedAt.Time, retrievedUser.PasswordChangedAt.Time, time.Second)
	require.WithinDuration(t, user.CreatedAt.Time, retrievedUser.CreatedAt.Time, time.Second)
}