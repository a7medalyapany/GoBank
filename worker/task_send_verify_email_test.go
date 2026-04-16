package worker

import (
	"context"
	"errors"
	"testing"

	db "github.com/a7medalyapany/GoBank.git/db/sqlc"
	"github.com/a7medalyapany/GoBank.git/util"
	"github.com/stretchr/testify/require"
)

type mockEmailSender struct {
	err      error
	subjects []string
	contents []string
	to       [][]string
}

func (mock *mockEmailSender) SendEmail(
	subject string,
	content string,
	to []string,
	_ []string,
	_ []string,
	_ []string,
) error {
	mock.subjects = append(mock.subjects, subject)
	mock.contents = append(mock.contents, content)
	mock.to = append(mock.to, append([]string{}, to...))
	return mock.err
}

func createWorkerTestUser(t *testing.T) db.User {
	t.Helper()

	hashedPassword, err := util.HashPassword(util.RandomString(12))
	require.NoError(t, err)

	user, err := workerTestStore.CreateUser(context.Background(), db.CreateUserParams{
		Username:       util.RandomOwner(),
		HashedPassword: hashedPassword,
		FullName:       util.RandomOwner(),
		Email:          util.RandomEmail(),
	})
	require.NoError(t, err)

	return user
}

func countVerifyEmailsByUsername(t *testing.T, username string) int64 {
	t.Helper()

	var count int64
	err := workerTestDB.QueryRow(
		context.Background(),
		"SELECT count(*) FROM verify_emails WHERE username = $1",
		username,
	).Scan(&count)
	require.NoError(t, err)

	return count
}

func TestProcessTaskSendVerifyEmailForUserReusesActiveVerifyEmailAfterRetry(t *testing.T) {
	ctx := context.Background()
	user := createWorkerTestUser(t)

	failingMailer := &mockEmailSender{err: errors.New("smtp unavailable")}
	handler := newVerifyEmailTaskHandler(workerTestStore, failingMailer, util.Config{BASE_URL: "http://localhost:3000"})

	err := handler.processTaskSendVerifyEmailForUser(ctx, user)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to send verification email")
	require.Equal(t, int64(1), countVerifyEmailsByUsername(t, user.Username))

	firstVerifyEmail, err := workerTestStore.GetLatestActiveVerifyEmail(ctx, db.GetLatestActiveVerifyEmailParams{
		Username: user.Username,
		Email:    user.Email,
	})
	require.NoError(t, err)

	successMailer := &mockEmailSender{}
	handler = newVerifyEmailTaskHandler(workerTestStore, successMailer, util.Config{BASE_URL: "http://localhost:3000"})

	err = handler.processTaskSendVerifyEmailForUser(ctx, user)
	require.NoError(t, err)
	require.Equal(t, int64(1), countVerifyEmailsByUsername(t, user.Username))
	require.Len(t, successMailer.contents, 1)

	secondVerifyEmail, err := workerTestStore.GetLatestActiveVerifyEmail(ctx, db.GetLatestActiveVerifyEmailParams{
		Username: user.Username,
		Email:    user.Email,
	})
	require.NoError(t, err)
	require.Equal(t, firstVerifyEmail.ID, secondVerifyEmail.ID)
	require.Equal(t, firstVerifyEmail.SecretCode, secondVerifyEmail.SecretCode)
}

func TestProcessTaskSendVerifyEmailForUserCreatesNewVerifyEmailWhenPreviousIsInactive(t *testing.T) {
	ctx := context.Background()
	user := createWorkerTestUser(t)

	mailer := &mockEmailSender{}
	handler := newVerifyEmailTaskHandler(workerTestStore, mailer, util.Config{BASE_URL: "http://localhost:3000"})

	err := handler.processTaskSendVerifyEmailForUser(ctx, user)
	require.NoError(t, err)
	require.Equal(t, int64(1), countVerifyEmailsByUsername(t, user.Username))

	firstVerifyEmail, err := workerTestStore.GetLatestActiveVerifyEmail(ctx, db.GetLatestActiveVerifyEmailParams{
		Username: user.Username,
		Email:    user.Email,
	})
	require.NoError(t, err)

	_, err = workerTestDB.Exec(
		ctx,
		"UPDATE verify_emails SET is_used = true, expires_at = now() - interval '1 minute' WHERE id = $1",
		firstVerifyEmail.ID,
	)
	require.NoError(t, err)

	err = handler.processTaskSendVerifyEmailForUser(ctx, user)
	require.NoError(t, err)
	require.Equal(t, int64(2), countVerifyEmailsByUsername(t, user.Username))

	latestVerifyEmail, err := workerTestStore.GetLatestActiveVerifyEmail(ctx, db.GetLatestActiveVerifyEmailParams{
		Username: user.Username,
		Email:    user.Email,
	})
	require.NoError(t, err)
	require.NotEqual(t, firstVerifyEmail.ID, latestVerifyEmail.ID)
}
