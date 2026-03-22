package gapi

import (
	"context"
	"errors"
	"time"

	db "github.com/a7medalyapany/GoBank.git/db/sqlc"
	"github.com/a7medalyapany/GoBank.git/pb"
	"github.com/a7medalyapany/GoBank.git/token"
	"github.com/a7medalyapany/GoBank.git/util"
	"github.com/a7medalyapany/GoBank.git/val"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)


func (server *Server) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	if violations := validateUpdateUserRequest(req); violations != nil {
		return nil, invalidArgumentError(violations)
	}

	authPayload, ok := ctx.Value(authPayloadKey).(*token.Payload)
	if !ok || authPayload == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	if authPayload.Username != req.GetUsername() {
		return nil, status.Errorf(codes.PermissionDenied, "cannot update another user's profile")
	}

	arg := db.UpdateUserParams{
		Username: req.GetUsername(),
	}

	if req.FullName != nil {
		arg.FullName = pgtype.Text{String: req.GetFullName(), Valid: true}
	}

	if req.Email != nil {
		arg.Email = pgtype.Text{String: req.GetEmail(), Valid: true}
	}

	if req.Password != nil {
		hashedPassword, err := util.HashPassword(req.GetPassword())
		if err != nil {
			return nil, status.Errorf(codes.Internal, "cannot hash password: %v", err)
		}
		arg.HashedPassword = pgtype.Text{String: hashedPassword, Valid: true}
		// stamp the time only when password actually changes
		arg.PasswordChangedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
	}

	user, err := server.store.UpdateUser(ctx, arg)
	if err != nil {
		// User was deleted between auth check and update (race condition)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "user not found")
		}

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505": // unique_violation
				return nil, status.Errorf(codes.AlreadyExists, "email already in use")
			}
		}

		return nil, status.Errorf(codes.Internal, "failed to update user: %v", err)
	}

	return &pb.UpdateUserResponse{User: convertUser(user)}, nil
}


func validateUpdateUserRequest(req *pb.UpdateUserRequest) (violations []*errdetails.BadRequest_FieldViolation) {
	// username is always required — it identifies who to update
	if err := val.ValidateUsername(req.GetUsername()); err != nil {
		violations = append(violations, fieldViolation("username", err))
	}

	// optional fields — only validate if the client actually sent them
	if req.FullName != nil {
		if err := val.ValidateFullname(req.GetFullName()); err != nil {
			violations = append(violations, fieldViolation("full_name", err))
		}
	}

	if req.Email != nil {
		if err := val.ValidateEmail(req.GetEmail()); err != nil {
			violations = append(violations, fieldViolation("email", err))
		}
	}

	if req.Password != nil {
		if err := val.ValidatePassword(req.GetPassword()); err != nil {
			violations = append(violations, fieldViolation("password", err))
		}
	}

	// Edge case: sending an update with no fields to change is a client mistake
	if req.FullName == nil && req.Email == nil && req.Password == nil {
		violations = append(violations, fieldViolation("body", errors.New("at least one field must be provided for update")))
	}

	return
}