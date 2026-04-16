package gapi

import (
	"context"
	"errors"
	"testing"
	"time"

	db "github.com/a7medalyapany/GoBank.git/db/sqlc"
	"github.com/a7medalyapany/GoBank.git/pb"
	"github.com/a7medalyapany/GoBank.git/util"
	"github.com/a7medalyapany/GoBank.git/worker"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"
)

type stubTaskDistributor struct {
	err      error
	payloads []*worker.PayloadSendVerifyEmail
	opts     [][]interface{}
}

func (stub *stubTaskDistributor) DistributeTaskSendVerifyEmail(
	_ context.Context,
	payload *worker.PayloadSendVerifyEmail,
	opts ...interface{},
) error {
	copiedPayload := *payload
	copiedOpts := append([]interface{}{}, opts...)

	stub.payloads = append(stub.payloads, &copiedPayload)
	stub.opts = append(stub.opts, copiedOpts)

	return stub.err
}

func newCreateUserTestServer(t *testing.T, taskDistributor worker.TaskDistributor) *Server {
	t.Helper()

	server, err := NewServer(testStore, util.Config{
		TOKEN_SYMMETRIC_KEY:   util.RandomString(32),
		ACCESS_TOKEN_DURATION: time.Minute,
	}, taskDistributor)
	require.NoError(t, err)

	return server
}

func randomCreateUserRequest() *pb.CreateUserRequest {
	return &pb.CreateUserRequest{
		Username: util.RandomOwner(),
		Password: util.RandomString(12),
		FullName: "John Doe",
		Email:    util.RandomEmail(),
	}
}

func TestCreateUserDispatchFailureDoesNotFailRegistration(t *testing.T) {
	distributor := &stubTaskDistributor{err: errors.New("dispatch failed")}
	server := newCreateUserTestServer(t, distributor)
	ctx := context.Background()
	req := randomCreateUserRequest()

	resp, err := server.CreateUser(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.User)
	require.Equal(t, req.Username, resp.User.Username)

	require.Len(t, distributor.payloads, 1)
	require.Equal(t, req.Username, distributor.payloads[0].Username)
	require.Len(t, distributor.opts, 1)
	require.Len(t, distributor.opts[0], 3)

	storedUser, err := testStore.GetUser(ctx, req.Username)
	require.NoError(t, err)
	require.Equal(t, req.Email, storedUser.Email)
}

func TestCreateUserDispatchesVerifyEmailTaskWithOptions(t *testing.T) {
	distributor := &stubTaskDistributor{}
	server := newCreateUserTestServer(t, distributor)
	req := randomCreateUserRequest()

	resp, err := server.CreateUser(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	require.Len(t, distributor.payloads, 1)
	require.Equal(t, req.Username, distributor.payloads[0].Username)
	require.Len(t, distributor.opts, 1)
	require.Len(t, distributor.opts[0], 3)

	for _, opt := range distributor.opts[0] {
		_, ok := opt.(asynq.Option)
		require.True(t, ok)
	}
}

func TestCreateUserWithNilTaskDistributor(t *testing.T) {
	server := newCreateUserTestServer(t, nil)
	req := randomCreateUserRequest()

	resp, err := server.CreateUser(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	storedUser, err := testStore.GetUser(context.Background(), req.Username)
	require.NoError(t, err)
	require.Equal(t, req.Username, storedUser.Username)
}

func TestCreateUserHandlesDuplicateUsernameOrEmail(t *testing.T) {
	distributor := &stubTaskDistributor{}
	server := newCreateUserTestServer(t, distributor)

	hashedPassword, err := util.HashPassword(util.RandomString(12))
	require.NoError(t, err)

	existingUser, err := testStore.CreateUser(context.Background(), db.CreateUserParams{
		Username:       util.RandomOwner(),
		HashedPassword: hashedPassword,
		FullName:       util.RandomOwner(),
		Email:          util.RandomEmail(),
	})
	require.NoError(t, err)

	resp, err := server.CreateUser(context.Background(), &pb.CreateUserRequest{
		Username: existingUser.Username,
		Password: util.RandomString(12),
		FullName: "Jane Doe",
		Email:    existingUser.Email,
	})
	require.Nil(t, resp)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already exists")
	require.Empty(t, distributor.payloads)
}
