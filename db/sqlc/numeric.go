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

// NumericToInt64 converts pgtype.Numeric to int64 with a specific exponent
// Example: NumericToInt64(numeric, -2) returns value in cents
// Uses big.Int arithmetic to avoid truncation issues
func NumericToInt64(n pgtype.Numeric, targetExp int32) int64 {
	if !n.Valid || n.Int == nil {
		return 0
	}

	// Use big.Int for arbitrary precision to avoid truncation
	result := new(big.Int).Set(n.Int)
	exponentDiff := targetExp - n.Exp

	if exponentDiff > 0 {
		// Need to multiply (add decimal places)
		// e.g., converting 1.5 (Exp=-1) to cents (Exp=-2): multiply by 10
		multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(exponentDiff)), nil)
		result.Mul(result, multiplier)
	} else if exponentDiff < 0 {
		// Need to divide (remove decimal places)
		// e.g., converting 150 cents (Exp=-2) to dollars (Exp=0): divide by 100
		divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-exponentDiff)), nil)
		result.Div(result, divisor)
	}

	return result.Int64()
}

// CompareNumeric compares two pgtype.Numeric values
// Returns: -1 if a < b, 0 if a == b, 1 if a > b
func CompareNumeric(a, b pgtype.Numeric) int {
	if !a.Valid || !b.Valid {
		return 0
	}

	// Convert both to the same exponent (use the smaller one for precision)
	targetExp := a.Exp
	if b.Exp < targetExp {
		targetExp = b.Exp
	}

	// Use big.Int for the comparison to handle large numbers
	aValue := new(big.Int)
	bValue := new(big.Int)

	// Convert a to target exponent
	exponentDiffA := targetExp - a.Exp
	aValue.Set(a.Int)
	if exponentDiffA > 0 {
		multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(exponentDiffA)), nil)
		aValue.Mul(aValue, multiplier)
	} else if exponentDiffA < 0 {
		divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-exponentDiffA)), nil)
		aValue.Div(aValue, divisor)
	}

	// Convert b to target exponent
	exponentDiffB := targetExp - b.Exp
	bValue.Set(b.Int)
	if exponentDiffB > 0 {
		multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(exponentDiffB)), nil)
		bValue.Mul(bValue, multiplier)
	} else if exponentDiffB < 0 {
		divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-exponentDiffB)), nil)
		bValue.Div(bValue, divisor)
	}

	// Compare using big.Int.Cmp
	return aValue.Cmp(bValue)
}