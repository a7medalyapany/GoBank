package gapi

import (
	"context"
	"errors"

	db "github.com/a7medalyapany/GoBank.git/db/sqlc"
	"github.com/a7medalyapany/GoBank.git/pb"
	"github.com/a7medalyapany/GoBank.git/token"
	"github.com/a7medalyapany/GoBank.git/util"
	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (server *Server) CreateTransfer(ctx context.Context, req *pb.CreateTransferRequest) (*pb.CreateTransferResponse, error) {
	authPayload, ok := ctx.Value(authPayloadKey).(*token.Payload)
	if !ok || authPayload == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	// Validate source account — also verifies currency match
	fromAccount, err := server.validateTransferAccount(ctx, req.GetFromAccountId(), req.GetCurrency())
	if err != nil {
		return nil, err
	}

	// Ownership check — only the authenticated user may debit their own account
	if fromAccount.Owner != authPayload.Username {
		return nil, status.Errorf(codes.PermissionDenied, "from_account doesn't belong to the authenticated user")
	}

	// Validate destination account — currency must match
	if _, err = server.validateTransferAccount(ctx, req.GetToAccountId(), req.GetCurrency()); err != nil {
		return nil, err
	}

	amountCents := util.FloatToCents(req.GetAmount())

	// Balance check before attempting the transaction
	if fromAccount.Balance < amountCents {
		return nil, status.Errorf(codes.FailedPrecondition,
			"insufficient balance: account %d has %s but transfer requires %s",
			req.GetFromAccountId(),
			util.FormatMoney(fromAccount.Balance, fromAccount.Currency),
			util.FormatMoney(amountCents, req.GetCurrency()),
		)
	}

	result, err := server.store.TransferTx(ctx, db.TransferTxParams{
		FromAccountID: req.GetFromAccountId(),
		ToAccountID:   req.GetToAccountId(),
		Amount:        amountCents,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "transfer transaction failed: %v", err)
	}

	return convertTransferResult(result), nil
}

// validateTransferAccount fetches an account and confirms its currency matches.
func (server *Server) validateTransferAccount(ctx context.Context, accountID int64, currency string) (db.Account, error) {
	account, err := server.store.GetAccount(ctx, accountID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Account{}, status.Errorf(codes.NotFound, "account %d not found", accountID)
		}
		return db.Account{}, status.Errorf(codes.Internal, "failed to get account %d: %v", accountID, err)
	}

	if account.Currency != currency {
		return db.Account{}, status.Errorf(codes.InvalidArgument,
			"account %d currency mismatch: expected %s, got %s",
			accountID, currency, account.Currency,
		)
	}

	return account, nil
}

// convertTransferResult maps a db.TransferTxResult to the pb wire type.
func convertTransferResult(r db.TransferTxResult) *pb.CreateTransferResponse {
	return &pb.CreateTransferResponse{
		Transfer: &pb.TransferRecord{
			Id:            r.Transfer.ID,
			FromAccountId: r.Transfer.FromAccountID,
			ToAccountId:   r.Transfer.ToAccountID,
			Amount:        util.CentsToFloat(r.Transfer.Amount),
			CreatedAt:     timestamppb.New(r.Transfer.CreatedAt.Time),
		},
		FromEntry: &pb.TransferEntry{
			Id:        r.FromEntry.ID,
			AccountId: r.FromEntry.AccountID,
			Amount:    util.CentsToFloat(r.FromEntry.Amount),
			CreatedAt: timestamppb.New(r.FromEntry.CreatedAt.Time),
		},
		ToEntry: &pb.TransferEntry{
			Id:        r.ToEntry.ID,
			AccountId: r.ToEntry.AccountID,
			Amount:    util.CentsToFloat(r.ToEntry.Amount),
			CreatedAt: timestamppb.New(r.ToEntry.CreatedAt.Time),
		},
		FromAccount: &pb.CreateTransferResponse_AccountSnapshot{
			Id:       r.FromAccount.ID,
			Owner:    r.FromAccount.Owner,
			Balance:  util.CentsToFloat(r.FromAccount.Balance),
			Currency: r.FromAccount.Currency,
		},
		ToAccount: &pb.CreateTransferResponse_AccountSnapshot{
			Id:       r.ToAccount.ID,
			Owner:    r.ToAccount.Owner,
			Balance:  util.CentsToFloat(r.ToAccount.Balance),
			Currency: r.ToAccount.Currency,
		},
	}
}
