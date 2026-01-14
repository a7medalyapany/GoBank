package db

import (
	"context"
	"testing"
	"time"

	"github.com/a7medalyapany/GoBank.git/util"
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