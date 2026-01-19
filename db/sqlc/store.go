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


// TransferTxResult is the result of the transfer transaction
type TransferTxParams struct {
	FromAccountID int64 `json:"from_account_id"`
	ToAccountID   int64 `json:"to_account_id"`
	Amount        pgtype.Numeric `json:"amount"`
}

type TransferTxResult struct {
	Transfer    Transfer `json:"transfer"`
	FromAccount Account  `json:"from_account"`
	ToAccount   Account  `json:"to_account"`
	FromEntry Entry `json:"from_entry"`
	ToEntry   Entry `json:"to_entry"`
}

// TransferTx performs a money transfer from one account to another.
// It creates a transfer record, add account entries, and update accounts' balance within a single db transaction
func (store *Store) TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error) {
	var result TransferTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		// create a transfer record
		result.Transfer, err = q.CreateTransfer(ctx, 
			CreateTransferParams{
				arg.FromAccountID,
				arg.ToAccountID,
				arg.Amount,
			},
		)
		if err != nil {
			return err
		}

		//add entries
		negatedAmount, err := NegateNumeric(arg.Amount)
		if err != nil {
			return err
		}

		result.FromEntry, err = q.CreateEntry(ctx,
			CreateEntryParams{
				AccountID: arg.FromAccountID,
				Amount:    negatedAmount,
			},
		)
		if err != nil {
			return err
		}


		result.ToEntry, err = q.CreateEntry(ctx,
			CreateEntryParams{
				AccountID: arg.ToAccountID,
				Amount:    arg.Amount,
			},
		)
		if err != nil {
			return err
		}

		//TODO: update accounts' balance


		result.FromAccount, err = q.AddAccountBalance(ctx,
			AddAccountBalanceParams{
				ID:     arg.FromAccountID,
				Amount: negatedAmount,
			},
		)
		if err != nil {
			return err
		}

		result.ToAccount, err = q.AddAccountBalance(ctx,
			AddAccountBalanceParams{
				ID:      arg.ToAccountID,
				Amount:  arg.Amount,
			},
		)
		if err != nil {
			return err
		}
		


		return nil
	})

	return result, err
}