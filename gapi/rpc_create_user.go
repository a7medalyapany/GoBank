package gapi

import (
	"context"
	"errors"
	"log"
	"time"

	db "github.com/a7medalyapany/GoBank.git/db/sqlc"
	"github.com/a7medalyapany/GoBank.git/pb"
	"github.com/a7medalyapany/GoBank.git/util"
	"github.com/a7medalyapany/GoBank.git/val"
	"github.com/a7medalyapany/GoBank.git/worker"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (server *Server) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	if violations := validateCreateUserRequest(req); violations != nil {
		return nil, invalidArgumentError(violations)
	}

	hashedPassword, err := util.HashPassword(req.GetPassword())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot hash password: %v", err)
	}

	arg := db.CreateUserTxParams{
		CreateUserParams: db.CreateUserParams{
			Username:       req.GetUsername(),
			HashedPassword: hashedPassword,
			FullName:       req.GetFullName(),
			Email:          req.GetEmail(),
		},
	}

	txResult, err := server.store.CreateUserTx(ctx, arg)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505":
				return nil, status.Errorf(codes.AlreadyExists, "username or email already exists: %v", err)
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}

	if err := server.dispatchVerifyEmailTask(ctx, txResult.User); err != nil {
		log.Printf("warning: failed to dispatch verify email task for user %s: %v", txResult.User.Username, err)
	}

	return &pb.CreateUserResponse{User: convertUser(txResult.User)}, nil
}

func (server *Server) dispatchVerifyEmailTask(ctx context.Context, user db.User) error {
	if server.taskDistributor == nil {
		return nil
	}

	taskPayload := &worker.PayloadSendVerifyEmail{
		Username: user.Username,
	}

	opts := []asynq.Option{
		asynq.MaxRetry(10),
		asynq.ProcessIn(10 * time.Second),
		asynq.Queue(worker.QueueCritical),
	}

	taskOpts := make([]interface{}, 0, len(opts))
	for _, opt := range opts {
		taskOpts = append(taskOpts, opt)
	}

	return server.taskDistributor.DistributeTaskSendVerifyEmail(ctx, taskPayload, taskOpts...)
}

func validateCreateUserRequest(req *pb.CreateUserRequest) (violations []*errdetails.BadRequest_FieldViolation) {
	if err := val.ValidateUsername(req.GetUsername()); err != nil {
		violations = append(violations, fieldViolation("username", err))
	}
	if err := val.ValidatePassword(req.GetPassword()); err != nil {
		violations = append(violations, fieldViolation("password", err))
	}
	if err := val.ValidateEmail(req.GetEmail()); err != nil {
		violations = append(violations, fieldViolation("email", err))
	}
	if err := val.ValidateFullname(req.GetFullName()); err != nil {
		violations = append(violations, fieldViolation("full_name", err))
	}
	return
}
