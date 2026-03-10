package gapi

import (
	"context"
	"errors"

	db "github.com/a7medalyapany/GoBank.git/db/sqlc"
	"github.com/a7medalyapany/GoBank.git/pb"
	"github.com/a7medalyapany/GoBank.git/token"
	"github.com/a7medalyapany/GoBank.git/util"
	"github.com/a7medalyapany/GoBank.git/val"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func convertAccount(a db.Account) *pb.Account {
	return &pb.Account{
		Id:        a.ID,
		Owner:     a.Owner,
		Balance:   util.CentsToFloat(a.Balance),
		Currency:  a.Currency,
		CreatedAt: timestamppb.New(a.CreatedAt.Time),
	}
}

func (server *Server) authorizeAccount(ctx context.Context, accountID int64) (db.Account, error) {
	account, err := server.store.GetAccount(ctx, accountID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Account{}, status.Errorf(codes.NotFound, "account not found")
		}
		return db.Account{}, status.Errorf(codes.Internal, "failed to get account: %v", err)
	}

	authPayload, ok := ctx.Value(authPayloadKey).(*token.Payload)
	if !ok || authPayload == nil {
		return db.Account{}, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	if account.Owner != authPayload.Username {
		return db.Account{}, status.Errorf(codes.PermissionDenied, "account doesn't belong to the authenticated user")
	}

	return account, nil
}

// CreateAccount
func (server *Server) CreateAccount(ctx context.Context, req *pb.CreateAccountRequest) (*pb.CreateAccountResponse, error) {
	if violations := validateCreateAccountRequest(req); violations != nil {
		return nil, invalidArgumentError(violations)
	}

	authPayload, ok := ctx.Value(authPayloadKey).(*token.Payload)
	if !ok || authPayload == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	account, err := server.store.CreateAccount(ctx, db.CreateAccountParams{
		Owner:    authPayload.Username,
		Currency: req.GetCurrency(),
		Balance:  0,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23503", "23505":
				return nil, status.Errorf(codes.AlreadyExists, "account already exists for this currency")
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to create account: %v", err)
	}

	return &pb.CreateAccountResponse{Account: convertAccount(account)}, nil
}

func validateCreateAccountRequest(req *pb.CreateAccountRequest) (violations []*errdetails.BadRequest_FieldViolation) {
	if err := val.ValidateCurrency(req.GetCurrency()); err != nil {
		violations = append(violations, fieldViolation("currency", err))
	}
	return
}

// GetAccount
func (server *Server) GetAccount(ctx context.Context, req *pb.GetAccountRequest) (*pb.GetAccountResponse, error) {
	if violations := validateGetAccountRequest(req); violations != nil {
		return nil, invalidArgumentError(violations)
	}

	account, err := server.authorizeAccount(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	return &pb.GetAccountResponse{Account: convertAccount(account)}, nil
}

func validateGetAccountRequest(req *pb.GetAccountRequest) (violations []*errdetails.BadRequest_FieldViolation) {
	if err := val.ValidateID(req.GetId()); err != nil {
		violations = append(violations, fieldViolation("id", err))
	}
	return
}

// ListAccounts
func (server *Server) ListAccounts(ctx context.Context, req *pb.ListAccountsRequest) (*pb.ListAccountsResponse, error) {
	if violations := validateListAccountsRequest(req); violations != nil {
		return nil, invalidArgumentError(violations)
	}

	authPayload, ok := ctx.Value(authPayloadKey).(*token.Payload)
	if !ok || authPayload == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	accounts, err := server.store.ListAccounts(ctx, db.ListAccountsParams{
		Owner:  authPayload.Username,
		Limit:  req.GetPageSize(),
		Offset: (req.GetPageId() - 1) * req.GetPageSize(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list accounts: %v", err)
	}

	pbAccounts := make([]*pb.Account, len(accounts))
	for i, a := range accounts {
		pbAccounts[i] = convertAccount(a)
	}

	return &pb.ListAccountsResponse{Accounts: pbAccounts}, nil
}

func validateListAccountsRequest(req *pb.ListAccountsRequest) (violations []*errdetails.BadRequest_FieldViolation) {
	if err := val.ValidatePageID(req.GetPageId()); err != nil {
		violations = append(violations, fieldViolation("page_id", err))
	}
	if err := val.ValidatePageSize(req.GetPageSize()); err != nil {
		violations = append(violations, fieldViolation("page_size", err))
	}
	return
}

// UpdateAccount
func (server *Server) UpdateAccount(ctx context.Context, req *pb.UpdateAccountRequest) (*pb.UpdateAccountResponse, error) {
	if violations := validateUpdateAccountRequest(req); violations != nil {
		return nil, invalidArgumentError(violations)
	}

	if _, err := server.authorizeAccount(ctx, req.GetId()); err != nil {
		return nil, err
	}

	updated, err := server.store.UpdateAccount(ctx, db.UpdateAccountParams{
		ID:      req.GetId(),
		Balance: util.FloatToCents(req.GetBalance()),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "account not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to update account: %v", err)
	}

	return &pb.UpdateAccountResponse{Account: convertAccount(updated)}, nil
}

func validateUpdateAccountRequest(req *pb.UpdateAccountRequest) (violations []*errdetails.BadRequest_FieldViolation) {
	if err := val.ValidateID(req.GetId()); err != nil {
		violations = append(violations, fieldViolation("id", err))
	}
	if req.GetBalance() < 0 {
		violations = append(violations, fieldViolation("balance", errors.New("must be non-negative")))
	}
	return
}

// DeleteAccount
func (server *Server) DeleteAccount(ctx context.Context, req *pb.DeleteAccountRequest) (*pb.DeleteAccountResponse, error) {
	if violations := validateDeleteAccountRequest(req); violations != nil {
		return nil, invalidArgumentError(violations)
	}

	if _, err := server.authorizeAccount(ctx, req.GetId()); err != nil {
		return nil, err
	}

	if err := server.store.DeleteAccount(ctx, req.GetId()); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "account not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to delete account: %v", err)
	}

	return &pb.DeleteAccountResponse{Status: "deleted"}, nil
}

func validateDeleteAccountRequest(req *pb.DeleteAccountRequest) (violations []*errdetails.BadRequest_FieldViolation) {
	if err := val.ValidateID(req.GetId()); err != nil {
		violations = append(violations, fieldViolation("id", err))
	}
	return
}