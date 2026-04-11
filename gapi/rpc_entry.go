package gapi

import (
	"context"
	"fmt"

	db "github.com/a7medalyapany/GoBank.git/db/sqlc"
	"github.com/a7medalyapany/GoBank.git/pb"
	"github.com/a7medalyapany/GoBank.git/token"
	"github.com/a7medalyapany/GoBank.git/val"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func (server *Server) ListEntries(ctx context.Context, req *pb.ListEntriesRequest) (*pb.ListEntriesResponse, error) {
	if violations := validateListEntriesRequest(req); violations != nil {
		return nil, invalidArgumentError(violations)
	}

	authPayload, ok := ctx.Value(authPayloadKey).(*token.Payload)
	if !ok || authPayload == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	rows, err := server.store.ListActivityEntries(ctx, db.ListActivityEntriesParams{
		Owner:  authPayload.Username,
		LimitArg: req.GetPageSize(),
		OffsetArg: (req.GetPageId() - 1) * req.GetPageSize(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list activity entries: %v", err)
	}

	entries := make([]*pb.ActivityEntry, len(rows))
	for i, row := range rows {
		entries[i] = convertActivityEntry(row)
	}

	return &pb.ListEntriesResponse{Entries: entries}, nil
}

func validateListEntriesRequest(req *pb.ListEntriesRequest) (violations []*errdetails.BadRequest_FieldViolation) {
	if err := val.ValidatePageID(req.GetPageId()); err != nil {
		violations = append(violations, fieldViolation("page_id", err))
	}
	if err := validateEntriesPageSize(req.GetPageSize()); err != nil {
		violations = append(violations, fieldViolation("page_size", err))
	}
	return
}

func validateEntriesPageSize(pageSize int32) error {
	if pageSize < 1 || pageSize > 50 {
		return fmt.Errorf("must be between 1 and 50")
	}
	return nil
}

func convertActivityEntry(row db.ListActivityEntriesRow) *pb.ActivityEntry {
	entry := &pb.ActivityEntry{
		Id:        row.ID,
		AccountId: row.AccountID,
		Amount:    row.Amount,
		Currency:  row.Currency,
		CreatedAt: timestamppb.New(row.CreatedAt.Time),
	}

	if row.TransferID.Valid {
		entry.TransferId = wrapperspb.Int64(row.TransferID.Int64)
	}
	if row.CounterpartAccountID.Valid {
		entry.CounterpartAccountId = wrapperspb.Int64(row.CounterpartAccountID.Int64)
	}
	if row.CounterpartOwner.Valid {
		entry.CounterpartOwner = wrapperspb.String(row.CounterpartOwner.String)
	}
	if row.CounterpartCurrency.Valid {
		entry.CounterpartCurrency = wrapperspb.String(row.CounterpartCurrency.String)
	}

	return entry
}
