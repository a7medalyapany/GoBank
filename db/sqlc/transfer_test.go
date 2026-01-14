package db

import (
	"context"
	"testing"
	"time"

	"github.com/a7medalyapany/GoBank.git/util"
	"github.com/stretchr/testify/require"
)


func createRandomTransfer(t *testing.T) Transfer {
	arg := CreateTransferParams{
		FromAccountID: createRandomAccount(t).ID,
		ToAccountID:   createRandomAccount(t).ID,
		Amount:        util.RandomMoney(),
	}

	transfer, err := testQueries.CreateTransfer(context.Background(), arg)

	require.NoError(t, err)
	require.NotEmpty(t, transfer)

	require.Equal(t, arg.FromAccountID, transfer.FromAccountID)
	require.Equal(t, arg.ToAccountID, transfer.ToAccountID)
	require.Equal(t, arg.Amount, transfer.Amount)
	require.NotZero(t, transfer.ID)
	require.NotZero(t, transfer.CreatedAt)
	return transfer
}

func TestCreateTransfer(t *testing.T) {
	createRandomTransfer(t)
}

func TestGetTransfer(t *testing.T) {
	transfer := createRandomTransfer(t)

	retrievedTransfer, err := testQueries.GetTransfer(context.Background(), transfer.ID)
	require.NoError(t, err)
	require.NotEmpty(t, retrievedTransfer)

	require.Equal(t, transfer.ID, retrievedTransfer.ID)
	require.Equal(t, transfer.FromAccountID, retrievedTransfer.FromAccountID)
	require.Equal(t, transfer.ToAccountID, retrievedTransfer.ToAccountID)
	require.Equal(t, transfer.Amount, retrievedTransfer.Amount)
	require.WithinDuration(t, transfer.CreatedAt.Time, retrievedTransfer.CreatedAt.Time, time.Second)
}
