package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store provides all functions to execute db queries and transactions
type Store struct {
	*Queries
	db *pgxpool.Pool
}

// NewStore creates a new Store
func NewStore(db *pgxpool.Pool) *Store {
	return &Store{
		Queries: New(db),
		db:      db,
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

	q := store.Queries.WithTx(tx)

	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("tx error: %w, rollback error: %w", err, rbErr)
		}
		return err
	}

	return tx.Commit(ctx)
}