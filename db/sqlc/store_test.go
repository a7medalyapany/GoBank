package db

import (
	"context"
	"math/big"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)


func TestTransferTx(t *testing.T) {
	store := NewStore(testDB)

	account1 := createRandomAccount(t)
	account2 := createRandomAccount(t)

	// Print balances before transfer
	t.Logf(
	">> Before transfer: Account1 = %s, Account2 = %s",
	FormatMoney(account1.Balance),
	FormatMoney(account2.Balance),
)


	// Create transfer amount (e.g., 10.00)
	amount := pgtype.Numeric{
		Int:   big.NewInt(1000), // 10.00 with 2 decimal places
		Exp:   -2,
		Valid: true,
	}

	// Run n concurrent transfer transactions
	n := 5
	errs := make(chan error)
	results := make(chan TransferTxResult)

	/*
	```
	### **Why test with multiple concurrent transfers?**

	In a **real banking system**, you might have:
	- User A transfers $10 to User B
	- **At the exact same time**, User C also transfers $10 to User B
	- Or worse: User A transfers to B, while B transfers to A **simultaneously**

	**This creates potential problems:**

	1. **Race Conditions** - Two transactions trying to update the same account balance at once
	2. **Deadlocks** - Transaction 1 locks Account A then waits for Account B, while Transaction 2 locks Account B and waits for Account A
	3. **Lost Updates** - Without proper isolation, balance changes might overwrite each other

	### **Visual Example of what `n := 5` does:**

	Imagine Account1 has $100 and Account2 has $50.

	**Without concurrency (n=1):**
	```
	Transfer $10 from Account1 → Account2

	Account1: $100 → $90
	Account2: $50  → $60
	```

	**With concurrency (n=5) - 5 transfers happening simultaneously:**
	```
	Time 0ms:  5 goroutines all start at the same time
			All trying to transfer $10 from Account1 → Account2

	Goroutine 1: Transfer $10 ━━━┐
	Goroutine 2: Transfer $10 ━━━┤
	Goroutine 3: Transfer $10 ━━━┼━━> Database must handle these correctly!
	Goroutine 4: Transfer $10 ━━━┤
	Goroutine 5: Transfer $10 ━━━┘

	Expected Result:
	Account1: $100 - (5 × $10) = $50
	Account2: $50  + (5 × $10) = $100
	*/

	for i := 0; i < n; i++ {
		go func() {
			ctx := context.Background()
			result, err := store.TransferTx(ctx, TransferTxParams{
				FromAccountID: account1.ID,
				ToAccountID: account2.ID,
				Amount: amount,
			})

			errs <- err
			results <- result
		}()
	}
	/*
	```
	**What's happening:**
	- `for i := 0; i < 5` - Loop 5 times
	- `go func()` - Launch a **goroutine** (lightweight thread) for each iteration
	- Each goroutine runs `TransferTx()` **independently and simultaneously**
	- `errs <- err` - Send the error (if any) to the channel
	- `results <- result` - Send the result to the channel

	**Visualization:**
	```
	Main Thread
		│
		├─> Goroutine 1: TransferTx($10) ━━━┐
		├─> Goroutine 2: TransferTx($10) ━━━┤
		├─> Goroutine 3: TransferTx($10) ━━━┼━━> All running at the same time!
		├─> Goroutine 4: TransferTx($10) ━━━┤
		└─> Goroutine 5: TransferTx($10) ━━━┘
	*/



	// Collect results

	existed := make(map[int64]bool)

	for i := 0; i < n; i++ {
		err := <- errs       // Wait for error from channel
		require.NoError(t, err)

		result := <- results // Wait for result from channel
		require.NotEmpty(t, result)

		// Additional checks can be added here to verify the correctness of each transfer

		// Check transfer record
		transfer := result.Transfer
		require.NotEmpty(t, transfer)
		require.Equal(t, transfer.FromAccountID, account1.ID)
		require.Equal(t, transfer.ToAccountID, account2.ID)
		require.Equal(t, amount.Int.String(), transfer.Amount.Int.String())
		require.NotZero(t, transfer.ID)
		require.NotZero(t, transfer.CreatedAt)

		_, err = store.queries.GetTransfer(context.Background(), transfer.ID)
		require.NoError(t, err)

		// check entries

		// From entry
		fromEntry := result.FromEntry
		require.NotEmpty(t, fromEntry)

		negatedAmount, err := NegateNumeric(amount)
		require.NoError(t, err)

		require.Equal(t, fromEntry.AccountID, account1.ID)
		require.Equal(t, negatedAmount.Int.String(), fromEntry.Amount.Int.String())
		require.NotZero(t, fromEntry.ID)
		require.NotZero(t, fromEntry.CreatedAt)

		_, err = store.queries.GetEntry(context.Background(), fromEntry.ID)
		require.NoError(t, err)


		// To entry
		toEntry := result.ToEntry
		require.NotEmpty(t, toEntry)

		require.Equal(t, toEntry.AccountID, account2.ID)
		require.Equal(t, amount.Int.String(), toEntry.Amount.Int.String())
		require.NotZero(t, toEntry.ID)
		require.NotZero(t, toEntry.CreatedAt)

		_, err = store.queries.GetEntry(context.Background(), toEntry.ID)
		require.NoError(t, err)


		// Check accounts

		fromAccount := result.FromAccount
		require.NotEmpty(t, fromAccount)
		require.Equal(t, fromAccount.ID, account1.ID)

		toAccount := result.ToAccount
		require.NotEmpty(t, toAccount)
		require.Equal(t, toAccount.ID, account2.ID)


		// Check accounts' balances

		t.Logf(
			">> tx %v: from = %s, to = %s",
			i+1,
			FormatMoney(fromAccount.Balance),
			FormatMoney(toAccount.Balance),
		)

		transferred := account1.Balance.Int.Int64() - fromAccount.Balance.Int.Int64()
		require.Equal(t, transferred,
			toAccount.Balance.Int.Int64() - account2.Balance.Int.Int64(),
		)

		require.True(t, transferred > 0)
		require.True(t, transferred%amount.Int.Int64() == 0) // transferred is multiple of amount
		// 1 * amount, 2 * amount, 3 * amount, ..., n * amount

		k := transferred / amount.Int.Int64()
		require.True(t, k >= 1 && k <= int64(n))
		require.NotContains(t, existed, k)
		existed[k] = true
	}

	// Check final updated balances
	updatedAccount1, err := store.queries.GetAccount(context.Background(), account1.ID)
	require.NoError(t, err)

	updatedAccount2, err := store.queries.GetAccount(context.Background(), account2.ID)
	require.NoError(t, err)

	require.Equal(t,
		account1.Balance.Int.Int64() - int64(n)*amount.Int.Int64(),
		updatedAccount1.Balance.Int.Int64(),
	)

	require.Equal(t,
		account2.Balance.Int.Int64() + int64(n)*amount.Int.Int64(),
		updatedAccount2.Balance.Int.Int64(),
	)
	
	// Print balances after transfer
	t.Logf(
		">> After transfer: Account1 = %s, Account2 = %s",
		FormatMoney(updatedAccount1.Balance),
		FormatMoney(updatedAccount2.Balance),
	)
}