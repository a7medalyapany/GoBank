package gapi

import (
	"context"
	"testing"
	"time"

	db "github.com/a7medalyapany/GoBank.git/db/sqlc"
	"github.com/a7medalyapany/GoBank.git/pb"
	"github.com/a7medalyapany/GoBank.git/token"
	"github.com/a7medalyapany/GoBank.git/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestListEntries(t *testing.T) {
	server := newTestServer(t)

	t.Run("Unauthenticated", func(t *testing.T) {
		resp, err := server.ListEntries(context.Background(), &pb.ListEntriesRequest{
			PageId:   1,
			PageSize: 5,
		})
		require.Nil(t, resp)
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("InvalidPageSize", func(t *testing.T) {
		ctx := authContext(t, createActivityFixture(t).user.Username)

		resp, err := server.ListEntries(ctx, &pb.ListEntriesRequest{
			PageId:   1,
			PageSize: 51,
		})
		require.Nil(t, resp)
		require.Error(t, err)
		require.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("ListsAuthenticatedUserEntriesOnly", func(t *testing.T) {
		fixture := createActivityFixture(t)
		ctx := authContext(t, fixture.user.Username)

		resp, err := server.ListEntries(ctx, &pb.ListEntriesRequest{
			PageId:   1,
			PageSize: 5,
		})
		require.NoError(t, err)
		require.Len(t, resp.Entries, 2)

		manual := resp.Entries[0]
		require.Equal(t, fixture.manualEntry.ID, manual.Id)
		require.Equal(t, fixture.userEUR.ID, manual.AccountId)
		require.Equal(t, int64(12345), manual.Amount)
		require.Equal(t, "EUR", manual.Currency)
		require.Nil(t, manual.TransferId)
		require.Nil(t, manual.CounterpartOwner)

		transferEntry := resp.Entries[1]
		require.Equal(t, fixture.transferResult.ToEntry.ID, transferEntry.Id)
		require.Equal(t, fixture.transferResult.Transfer.ID, transferEntry.TransferId.Value)
		require.Equal(t, fixture.counterpartyUSD.ID, transferEntry.CounterpartAccountId.Value)
		require.Equal(t, fixture.counterparty.Username, transferEntry.CounterpartOwner.Value)
		require.Equal(t, "USD", transferEntry.CounterpartCurrency.Value)

		body, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(resp)
		require.NoError(t, err)
		jsonBody := string(body)
		require.Contains(t, jsonBody, `"transferId":null`)
		require.Contains(t, jsonBody, `"counterpartOwner":null`)
		require.NotContains(t, jsonBody, fixture.outsider.Username)
	})
}

type activityFixture struct {
	user            db.User
	counterparty    db.User
	outsider        db.User
	userEUR         db.Account
	counterpartyUSD db.Account
	manualEntry     db.Entry
	transferResult  db.TransferTxResult
}

func createActivityFixture(t *testing.T) activityFixture {
	t.Helper()

	ctx := context.Background()

	user := createTestUser(t)
	counterparty := createTestUser(t)
	outsider := createTestUser(t)

	userUSD := createTestAccount(t, user.Username, "USD", 500_000)
	userEUR := createTestAccount(t, user.Username, "EUR", 50_000)
	counterpartyUSD := createTestAccount(t, counterparty.Username, "USD", 500_000)
	outsiderUSD := createTestAccount(t, outsider.Username, "USD", 500_000)

	manualEntry, err := testStore.CreateEntry(ctx, db.CreateEntryParams{
		AccountID: userEUR.ID,
		Amount:    12_345,
	})
	require.NoError(t, err)

	transferResult, err := testStore.TransferTx(ctx, db.TransferTxParams{
		FromAccountID: counterpartyUSD.ID,
		ToAccountID:   userUSD.ID,
		Amount:        5_000,
	})
	require.NoError(t, err)

	outsiderEntry, err := testStore.CreateEntry(ctx, db.CreateEntryParams{
		AccountID: outsiderUSD.ID,
		Amount:    999,
	})
	require.NoError(t, err)

	base := time.Now().UTC().Truncate(time.Second)
	setTestEntryCreatedAt(t, manualEntry.ID, base)
	setTestEntryCreatedAt(t, transferResult.ToEntry.ID, base.Add(-1*time.Hour))
	setTestEntryCreatedAt(t, outsiderEntry.ID, base.Add(time.Hour))

	return activityFixture{
		user:            user,
		counterparty:    counterparty,
		outsider:        outsider,
		userEUR:         userEUR,
		counterpartyUSD: counterpartyUSD,
		manualEntry:     manualEntry,
		transferResult:  transferResult,
	}
}

func createTestUser(t *testing.T) db.User {
	t.Helper()

	hashedPassword, err := util.HashPassword(util.RandomOwner())
	require.NoError(t, err)

	user, err := testStore.CreateUser(context.Background(), db.CreateUserParams{
		Username:       util.RandomOwner(),
		HashedPassword: hashedPassword,
		FullName:       util.RandomOwner(),
		Email:          util.RandomEmail(),
	})
	require.NoError(t, err)

	return user
}

func createTestAccount(t *testing.T, owner string, currency string, balance int64) db.Account {
	t.Helper()

	account, err := testStore.CreateAccount(context.Background(), db.CreateAccountParams{
		Owner:    owner,
		Balance:  balance,
		Currency: currency,
	})
	require.NoError(t, err)

	return account
}

func authContext(t *testing.T, username string) context.Context {
	t.Helper()

	payload, err := token.NewPayload(username, time.Minute)
	require.NoError(t, err)

	return context.WithValue(context.Background(), authPayloadKey, payload)
}

func setTestEntryCreatedAt(t *testing.T, entryID int64, createdAt time.Time) {
	t.Helper()

	_, err := testDB.Exec(context.Background(),
		"UPDATE entries SET created_at = $2 WHERE id = $1",
		entryID,
		pgtype.Timestamptz{Time: createdAt, Valid: true},
	)
	require.NoError(t, err)
}
