package db

import (
	"context"
	"testing"
	"time"

	"github.com/a7medalyapany/GoBank.git/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func createRandomEntry(t *testing.T) Entry {
	arg := CreateEntryParams{
		AccountID: createRandomAccount(t).ID,
		Amount:    util.RandomMoney(),
	}

	entry, err := testQueries.CreateEntry(context.Background(), arg)

	require.NoError(t, err)
	require.NotEmpty(t, entry)

	require.Equal(t, arg.AccountID, entry.AccountID)
	require.Equal(t, arg.Amount, entry.Amount)
	require.False(t, entry.TransferID.Valid)

	require.NotZero(t, entry.ID)
	require.NotZero(t, entry.CreatedAt)
	return entry
}

func TestCreateEntry(t *testing.T) {
	createRandomEntry(t)
}

func TestGetEntry(t *testing.T) {
	entry := createRandomEntry(t)

	retrievedEntry, err := testQueries.GetEntry(context.Background(), entry.ID)
	require.NoError(t, err)
	require.NotEmpty(t, retrievedEntry)

	require.Equal(t, entry.ID, retrievedEntry.ID)
	require.Equal(t, entry.AccountID, retrievedEntry.AccountID)
	require.Equal(t, entry.Amount, retrievedEntry.Amount)
	require.Equal(t, entry.TransferID, retrievedEntry.TransferID)
	require.WithinDuration(t, entry.CreatedAt.Time, retrievedEntry.CreatedAt.Time, time.Second)
}

func TestListEntries(t *testing.T) {
	account := createRandomAccount(t)

	// Create multiple entries for the same account
	for range 10 {
		arg := CreateEntryParams{
			AccountID: account.ID,
			Amount:    util.RandomMoney(),
		}
		testQueries.CreateEntry(context.Background(), arg)
	}

	// Test listing with limit
	arg := ListEntriesParams{
		AccountID: account.ID,
		Limit:     5,
		Offset:    0,
	}

	entries, err := testQueries.ListEntries(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, entries)
	require.Len(t, entries, 5)

	// Verify all entries belong to the correct account
	for _, entry := range entries {
		require.Equal(t, account.ID, entry.AccountID)
		require.NotZero(t, entry.ID)
		require.NotZero(t, entry.Amount)
		require.NotZero(t, entry.CreatedAt)
	}

	// Test with offset
	arg = ListEntriesParams{
		AccountID: account.ID,
		Limit:     5,
		Offset:    5,
	}

	entries, err = testQueries.ListEntries(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, entries)
	require.Len(t, entries, 5)
}

func TestListActivityEntries(t *testing.T) {
	ctx := context.Background()
	store := NewStore(testDB)

	user := createRandomUser(t)
	counterparty := createRandomUser(t)
	outsider := createRandomUser(t)

	userUSD, err := testQueries.CreateAccount(ctx, CreateAccountParams{
		Owner:    user.Username,
		Balance:  500_000,
		Currency: "USD",
	})
	require.NoError(t, err)

	userEUR, err := testQueries.CreateAccount(ctx, CreateAccountParams{
		Owner:    user.Username,
		Balance:  75_000,
		Currency: "EUR",
	})
	require.NoError(t, err)

	counterpartyUSD, err := testQueries.CreateAccount(ctx, CreateAccountParams{
		Owner:    counterparty.Username,
		Balance:  500_000,
		Currency: "USD",
	})
	require.NoError(t, err)

	outsiderUSD, err := testQueries.CreateAccount(ctx, CreateAccountParams{
		Owner:    outsider.Username,
		Balance:  500_000,
		Currency: "USD",
	})
	require.NoError(t, err)

	manualLatest, err := testQueries.CreateEntry(ctx, CreateEntryParams{
		AccountID: userEUR.ID,
		Amount:    12_345,
	})
	require.NoError(t, err)

	debitResult, err := store.TransferTx(ctx, TransferTxParams{
		FromAccountID: userUSD.ID,
		ToAccountID:   counterpartyUSD.ID,
		Amount:        2_500,
	})
	require.NoError(t, err)

	creditResult, err := store.TransferTx(ctx, TransferTxParams{
		FromAccountID: counterpartyUSD.ID,
		ToAccountID:   userUSD.ID,
		Amount:        1_250,
	})
	require.NoError(t, err)

	manualOldest, err := testQueries.CreateEntry(ctx, CreateEntryParams{
		AccountID: userUSD.ID,
		Amount:    500,
	})
	require.NoError(t, err)

	outsiderEntry, err := testQueries.CreateEntry(ctx, CreateEntryParams{
		AccountID: outsiderUSD.ID,
		Amount:    999,
	})
	require.NoError(t, err)

	base := time.Now().UTC().Truncate(time.Second)
	setEntryCreatedAt(t, manualLatest.ID, base.Add(-1*time.Hour))
	setEntryCreatedAt(t, debitResult.FromEntry.ID, base.Add(-2*time.Hour))
	setEntryCreatedAt(t, creditResult.ToEntry.ID, base.Add(-3*time.Hour))
	setEntryCreatedAt(t, manualOldest.ID, base.Add(-4*time.Hour))
	setEntryCreatedAt(t, outsiderEntry.ID, base)

	entries, err := testQueries.ListActivityEntries(ctx, ListActivityEntriesParams{
		Owner:  user.Username,
		LimitArg:  10,
		OffsetArg: 0,
	})
	require.NoError(t, err)
	require.Len(t, entries, 4)

	require.Equal(t, manualLatest.ID, entries[0].ID)
	require.Equal(t, userEUR.ID, entries[0].AccountID)
	require.Equal(t, "EUR", entries[0].Currency)
	require.False(t, entries[0].TransferID.Valid)
	require.False(t, entries[0].CounterpartAccountID.Valid)
	require.False(t, entries[0].CounterpartOwner.Valid)
	require.False(t, entries[0].CounterpartCurrency.Valid)

	require.Equal(t, debitResult.FromEntry.ID, entries[1].ID)
	require.Equal(t, debitResult.Transfer.ID, entries[1].TransferID.Int64)
	require.Equal(t, counterpartyUSD.ID, entries[1].CounterpartAccountID.Int64)
	require.Equal(t, counterparty.Username, entries[1].CounterpartOwner.String)
	require.Equal(t, "USD", entries[1].CounterpartCurrency.String)

	require.Equal(t, creditResult.ToEntry.ID, entries[2].ID)
	require.Equal(t, creditResult.Transfer.ID, entries[2].TransferID.Int64)
	require.Equal(t, counterpartyUSD.ID, entries[2].CounterpartAccountID.Int64)
	require.Equal(t, counterparty.Username, entries[2].CounterpartOwner.String)
	require.Equal(t, "USD", entries[2].CounterpartCurrency.String)

	require.Equal(t, manualOldest.ID, entries[3].ID)
	require.Equal(t, userUSD.ID, entries[3].AccountID)
	require.False(t, entries[3].TransferID.Valid)

	page1, err := testQueries.ListActivityEntries(ctx, ListActivityEntriesParams{
		Owner:  user.Username,
		LimitArg:  2,
		OffsetArg: 0,
	})
	require.NoError(t, err)
	require.Len(t, page1, 2)
	require.Equal(t, manualLatest.ID, page1[0].ID)
	require.Equal(t, debitResult.FromEntry.ID, page1[1].ID)

	page2, err := testQueries.ListActivityEntries(ctx, ListActivityEntriesParams{
		Owner:  user.Username,
		LimitArg:  2,
		OffsetArg: 2,
	})
	require.NoError(t, err)
	require.Len(t, page2, 2)
	require.Equal(t, creditResult.ToEntry.ID, page2[0].ID)
	require.Equal(t, manualOldest.ID, page2[1].ID)
}

func setEntryCreatedAt(t *testing.T, entryID int64, createdAt time.Time) {
	t.Helper()

	_, err := testDB.Exec(context.Background(),
		"UPDATE entries SET created_at = $2 WHERE id = $1",
		entryID,
		pgtype.Timestamptz{Time: createdAt, Valid: true},
	)
	require.NoError(t, err)
}
