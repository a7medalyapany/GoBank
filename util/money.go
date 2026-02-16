package util

import (
	"fmt"
	"math"
)

// FloatToCents converts dollars (or any currency) to cents
// Example: 10.50 → 1050
func FloatToCents(amount float64) int64 {
	return int64(math.Round(amount * 100))
}

// CentsToFloat converts cents to dollars
// Example: 1050 → 10.50
func CentsToFloat(cents int64) float64 {
	return float64(cents) / 100.0
}

// FormatMoney formats cents as a currency string
// Example: FormatMoney(1050, "USD") → "$10.50"
func FormatMoney(cents int64, currency string) string {
	symbol := "$"
	switch currency {
	case "EUR":
		symbol = "€"
	case "EGP":
		symbol = "E£"
	}
	return fmt.Sprintf("%s%.2f", symbol, CentsToFloat(cents))
}