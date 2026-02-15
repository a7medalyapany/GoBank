package util

const (
	USD = "USD"
	EUR = "EUR"
	EGP = "EGP"
)


func IsSupportedCurrency(currency string) bool {
	switch currency {
	case USD, EUR, EGP:
		return true
	}
	return false
}