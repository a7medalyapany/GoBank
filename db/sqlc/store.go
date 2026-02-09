package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store provides all functions to execute db queries and transactions
type Store struct {
	db      *pgxpool.Pool
	queries *Queries
}

// NewStore creates a new Store
func NewStore(db *pgxpool.Pool) *Store {
	return &Store{
		db:      db,
		queries: New(db),
	}
}

// execTx runs a function within a database transaction
func (store *Store) execTx(
	ctx context.Context,
	fn func(*Queries) error,
) error {

	tx, err := store.db.Begin(ctx)
	if err != nil {
		return err
	}

	q := store.queries.WithTx(tx)

	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("tx error: %v, rollback error: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit(ctx)
}

// TransferTxParams is input for transfer
type TransferTxParams struct {
	FromAccountID int64          `json:"from_account_id"`
	ToAccountID   int64          `json:"to_account_id"`
	Amount        pgtype.Numeric `json:"amount"`
}

type TransferTxResult struct {
	Transfer    Transfer `json:"transfer"`
	FromAccount Account  `json:"from_account"`
	ToAccount   Account  `json:"to_account"`
	FromEntry   Entry    `json:"from_entry"`
	ToEntry     Entry    `json:"to_entry"`
}

// TransferTx performs a money transfer from one account to another
func (store *Store) TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error) {
	var result TransferTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		// create transfer
		result.Transfer, err = q.CreateTransfer(ctx, CreateTransferParams{
			arg.FromAccountID,
			arg.ToAccountID,
			arg.Amount,
		})
		if err != nil {
			return err
		}

		// create entries
		negatedAmount, err := NegateNumeric(arg.Amount)
		if err != nil {
			return err
		}

		result.FromEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.FromAccountID,
			Amount:    negatedAmount,
		})
		if err != nil {
			return err
		}

		result.ToEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.ToAccountID,
			Amount:    arg.Amount,
		})
		if err != nil {
			return err
		}

		// update balances in consistent order (deadlock safe)
		if arg.FromAccountID < arg.ToAccountID {
			result.FromAccount, result.ToAccount, err = addMoney(ctx, q, arg.FromAccountID, negatedAmount, arg.ToAccountID, arg.Amount)
		} else {
			result.ToAccount, result.FromAccount, err = addMoney(ctx, q, arg.ToAccountID, arg.Amount, arg.FromAccountID, negatedAmount)
		}

		return nil
	})

	return result, err
}

func addMoney(
	ctx context.Context,
	q *Queries,
	accountID1 int64,
	amount1 pgtype.Numeric,
	accountID2 int64,
	amount2 pgtype.Numeric,
) (account1 Account, account2 Account, err error) {
	// update balances in consistent order (deadlock safe)
	account1, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
		ID:     accountID1,
		Amount: amount1,
	})
	if err != nil {
		return
	}

	account2, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
		ID:     accountID2,
		Amount: amount2,
	})

	return
}