package db

import (
	"fmt"
	"math/big"
	"strings"

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


// FormatMoney converts a pgtype.Numeric to a string representation
// with decimal point based on its exponent.
func FormatMoney(n pgtype.Numeric) string {
	if !n.Valid || n.Int == nil {
		return "invalid"
	}

	// copy to avoid mutating original
	i := new(big.Int).Set(n.Int)

	if n.Exp >= 0 {
		i.Mul(i, big.NewInt(0).Exp(big.NewInt(10), big.NewInt(int64(n.Exp)), nil))
		return i.String()
	}

	exp := int(-n.Exp)
	s := i.String()

	if len(s) <= exp {
		s = "0." + strings.Repeat("0", exp-len(s)) + s
	} else {
		pos := len(s) - exp
		s = s[:pos] + "." + s[pos:]
	}

	return s
}
