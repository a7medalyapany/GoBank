package gapi

import (
	"context"
	"errors"

	db "github.com/a7medalyapany/GoBank.git/db/sqlc"
	"github.com/a7medalyapany/GoBank.git/pb"
	"github.com/a7medalyapany/GoBank.git/val"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (server *Server) VerifyEmail(ctx context.Context, req *pb.VerifyEmailRequest) (*pb.VerifyEmailResponse, error) {
    if violations := validateVerifyEmailRequest(req); violations != nil {
        return nil, invalidArgumentError(violations)
    }

    result, err := server.store.VerifyEmailTx(ctx, db.VerifyEmailTxParams{
        EmailId:    req.GetEmailId(),
        SecretCode: req.GetSecretCode(),
    })
    if err != nil {
        return nil, status.Errorf(codes.Internal, "failed to verify email: %v", err)
    }

    return &pb.VerifyEmailResponse{
        IsVerified: result.User.IsEmailVerified,
    }, nil
}

func validateVerifyEmailRequest(req *pb.VerifyEmailRequest) (violations []*errdetails.BadRequest_FieldViolation) {
    if err := val.ValidateID(req.GetEmailId()); err != nil {
        violations = append(violations, fieldViolation("email_id", err))
    }
    if err := val.ValidateString(req.GetSecretCode(), 32, 128); err != nil {
        violations = append(violations, fieldViolation("secret_code", errors.New("must not be empty")))
    }
    return
}