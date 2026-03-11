package db

import (
	"context"
	"testing"
	"time"

	"github.com/a7medalyapany/GoBank.git/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func createRandomUser(t *testing.T) User {
	hashedPassword, err := util.HashPassword(util.RandomOwner())
	require.NoError(t, err)

	arg := CreateUserParams{
		Username: util.RandomOwner(),
		HashedPassword: hashedPassword,
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

func TestUpdateUserFullName(t *testing.T) {
	oldUser := createRandomUser(t)

	newFullName := util.RandomOwner()
	updatedUser, err := testQueries.UpdateUser(context.Background(), UpdateUserParams{
		Username: oldUser.Username,
		FullName: pgtype.Text{String: newFullName, Valid: true},
	})

	require.NoError(t, err)
	require.NotEmpty(t, updatedUser)
	require.NotEqual(t, oldUser.FullName, updatedUser.FullName)
	require.Equal(t, newFullName, updatedUser.FullName)

	require.Equal(t, oldUser.Email, updatedUser.Email)
	require.Equal(t, oldUser.HashedPassword, updatedUser.HashedPassword)
}

func TestUpdateUserEmail(t *testing.T) {
	oldUser := createRandomUser(t)

	newEmail := util.RandomEmail()
	updatedUser, err := testQueries.UpdateUser(context.Background(), UpdateUserParams{
		Username: oldUser.Username,
		Email:    pgtype.Text{String: newEmail, Valid: true},
	})

	require.NoError(t, err)
	require.NotEmpty(t, updatedUser)
	require.NotEqual(t, oldUser.Email, updatedUser.Email)
	require.Equal(t, newEmail, updatedUser.Email)

	require.Equal(t, oldUser.FullName, updatedUser.FullName)
	require.Equal(t, oldUser.HashedPassword, updatedUser.HashedPassword)
}

func TestUpdateUserPassword(t *testing.T) {
	oldUser := createRandomUser(t)

	newHashedPassword, err := util.HashPassword(util.RandomOwner())
	require.NoError(t, err)
	updatedUser, err := testQueries.UpdateUser(context.Background(), UpdateUserParams{
		Username:       oldUser.Username,
		HashedPassword: pgtype.Text{String: newHashedPassword, Valid: true},
	})

	require.NoError(t, err)
	require.NotEmpty(t, updatedUser)
	require.NotEqual(t, oldUser.HashedPassword, updatedUser.HashedPassword)
	require.Equal(t, newHashedPassword, updatedUser.HashedPassword)

	require.Equal(t, oldUser.FullName, updatedUser.FullName)
	require.Equal(t, oldUser.Email, updatedUser.Email)
}


func TestUpdateUserAllFields(t *testing.T) {
	oldUser := createRandomUser(t)

	newFullName := util.RandomOwner()
	newEmail := util.RandomEmail()
	newHashedPassword, err := util.HashPassword(util.RandomOwner())
	require.NoError(t, err)

	updatedUser, err := testQueries.UpdateUser(context.Background(), UpdateUserParams{
		Username:       oldUser.Username,
		FullName:       pgtype.Text{String: newFullName, Valid: true},
		Email:          pgtype.Text{String: newEmail, Valid: true},
		HashedPassword: pgtype.Text{String: newHashedPassword, Valid: true},
	})

	require.NoError(t, err)
	require.NotEmpty(t, updatedUser)
	require.NotEqual(t, oldUser.FullName, updatedUser.FullName)
	require.Equal(t, newFullName, updatedUser.FullName)
	require.NotEqual(t, oldUser.Email, updatedUser.Email)
	require.Equal(t, newEmail, updatedUser.Email)
	require.NotEqual(t, oldUser.HashedPassword, updatedUser.HashedPassword)
	require.Equal(t, newHashedPassword, updatedUser.HashedPassword)
}

func TestUpdateUserPasswordChangedAt(t *testing.T) {
    oldUser := createRandomUser(t)

    require.True(t, oldUser.PasswordChangedAt.Time.IsZero())

    newHashedPassword, err := util.HashPassword(util.RandomOwner())
    require.NoError(t, err)

    updatedUser, err := testQueries.UpdateUser(context.Background(), UpdateUserParams{
        Username:           oldUser.Username,
        HashedPassword:     pgtype.Text{String: newHashedPassword, Valid: true},
        PasswordChangedAt:  pgtype.Timestamptz{Time: time.Now(), Valid: true},
    })

    require.NoError(t, err)
    require.False(t, updatedUser.PasswordChangedAt.Time.IsZero())
    require.NotEqual(t, oldUser.HashedPassword, updatedUser.HashedPassword)
    

    require.Equal(t, oldUser.FullName, updatedUser.FullName)
    require.Equal(t, oldUser.Email, updatedUser.Email)
}