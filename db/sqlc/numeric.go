package db

import (
	"fmt"
	"math/big"

	"github.com/jackc/pgx/v5/pgtype"
)

// NegateNumeric returns a new Numeric with the sign flipped.
// It does not mutate the input.
func NegateNumeric(n pgtype.Numeric) (pgtype.Numeric, error) {
	if !n.Valid {
		return pgtype.Numeric{}, fmt.Errorf("invalid numeric value")
	}

	// Create a new big.Int and negate it
	negatedInt := new(big.Int).Neg(n.Int)

	return pgtype.Numeric{
		Int:   negatedInt,
		Exp:   n.Exp,
		NaN:   n.NaN,
		Valid: n.Valid,
	}, nil
}