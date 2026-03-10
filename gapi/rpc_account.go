package gapi

import (
	"context"
	"errors"

	db "github.com/a7medalyapany/GoBank.git/db/sqlc"
	"github.com/a7medalyapany/GoBank.git/pb"
	"github.com/a7medalyapany/GoBank.git/token"
	"github.com/a7medalyapany/GoBank.git/util"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// convertAccount maps a db.Account to the pb.Account wire type.
// Balance is converted from cents (int64) → float64 for the response.
func convertAccount(a db.Account) *pb.Account {
	return &pb.Account{
		Id:        a.ID,
		Owner:     a.Owner,
		Balance:   util.CentsToFloat(a.Balance),
		Currency:  a.Currency,
		CreatedAt: timestamppb.New(a.CreatedAt.Time),
	}
}

// authorizeAccount fetches an account and verifies the authenticated user owns it.
// Returns the account on success, or a gRPC status error on failure.
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

// ── CreateAccount ─────────────────────────────────────────────────────────────

func (server *Server) CreateAccount(ctx context.Context, req *pb.CreateAccountRequest) (*pb.CreateAccountResponse, error) {
	authPayload, ok := ctx.Value(authPayloadKey).(*token.Payload)
	if !ok || authPayload == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	arg := db.CreateAccountParams{
		Owner:    authPayload.Username,
		Currency: req.GetCurrency(),
		Balance:  0,
	}

	account, err := server.store.CreateAccount(ctx, arg)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23503", "23505": // foreign key / unique violation
				return nil, status.Errorf(codes.AlreadyExists, "account already exists for this currency: %v", err)
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to create account: %v", err)
	}

	return &pb.CreateAccountResponse{Account: convertAccount(account)}, nil
}

// ── GetAccount ────────────────────────────────────────────────────────────────

func (server *Server) GetAccount(ctx context.Context, req *pb.GetAccountRequest) (*pb.GetAccountResponse, error) {
	account, err := server.authorizeAccount(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	return &pb.GetAccountResponse{Account: convertAccount(account)}, nil
}

// ── ListAccounts ──────────────────────────────────────────────────────────────

func (server *Server) ListAccounts(ctx context.Context, req *pb.ListAccountsRequest) (*pb.ListAccountsResponse, error) {
	authPayload, ok := ctx.Value(authPayloadKey).(*token.Payload)
	if !ok || authPayload == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	arg := db.ListAccountsParams{
		Owner:  authPayload.Username,
		Limit:  req.GetPageSize(),
		Offset: (req.GetPageId() - 1) * req.GetPageSize(),
	}

	accounts, err := server.store.ListAccounts(ctx, arg)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list accounts: %v", err)
	}

	pbAccounts := make([]*pb.Account, len(accounts))
	for i, a := range accounts {
		pbAccounts[i] = convertAccount(a)
	}

	return &pb.ListAccountsResponse{Accounts: pbAccounts}, nil
}

// ── UpdateAccount ─────────────────────────────────────────────────────────────

func (server *Server) UpdateAccount(ctx context.Context, req *pb.UpdateAccountRequest) (*pb.UpdateAccountResponse, error) {
	// Ownership check first — never mutate before verifying
	if _, err := server.authorizeAccount(ctx, req.GetId()); err != nil {
		return nil, err
	}

	arg := db.UpdateAccountParams{
		ID:      req.GetId(),
		Balance: util.FloatToCents(req.GetBalance()),
	}

	updated, err := server.store.UpdateAccount(ctx, arg)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "account not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to update account: %v", err)
	}

	return &pb.UpdateAccountResponse{Account: convertAccount(updated)}, nil
}

// ── DeleteAccount ─────────────────────────────────────────────────────────────

func (server *Server) DeleteAccount(ctx context.Context, req *pb.DeleteAccountRequest) (*pb.DeleteAccountResponse, error) {
	// Ownership check first
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
