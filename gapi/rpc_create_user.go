package gapi

import (
	"context"
	"errors"

	db "github.com/a7medalyapany/GoBank.git/db/sqlc"
	"github.com/a7medalyapany/GoBank.git/pb"
	"github.com/a7medalyapany/GoBank.git/util"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (server *Server) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {

	hashedPassword, err := util.HashPassword(req.GetPassword())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Cannot hash password: %v", err)
	}

	arg := db.CreateUserParams{
		Username: req.GetUsername(),
		HashedPassword: hashedPassword,
		FullName: req.GetFullName(),
		Email: req.GetEmail(),
	}

	user, err := server.store.CreateUser(ctx, arg)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			// Unique violation error code is "23505"
			switch pgErr.Code {
			case "23505":
				return nil, status.Errorf(codes.AlreadyExists, "Username already exists: %v", err)
			}
		}
		return nil, status.Errorf(codes.Internal, "Failed to create user: %v", err)
	}

	res := &pb.CreateUserResponse{
		User: convertUser(user),
	}

	return res, nil
}