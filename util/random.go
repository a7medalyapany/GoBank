package util

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}


var firstNames = []string{
	"ali", "maria", "sarah", "mike", "jeo", 
	"emma", "marmosh", "menna", "shahd", "nashi",
}

var lastNames = []string{
	"ahmed", "alyapany", "brown", "saif",
	"doe", "miller", "salah", "emam",
}

// RandomOwner generates a random owner name in the format "first.last123".
func RandomOwner() string {
	first := firstNames[rand.Intn(len(firstNames))]
	last := lastNames[rand.Intn(len(lastNames))]
	number := rand.Intn(1000)

	// e.g. ahmed.alyapany69
	return strings.ToLower(
		first + "." + last + "." + RandomString(3) + string(rune('0'+number%10)),
	)
}


// RandomCurrency returns a random currency code from a predefined list, with a bias towards USD.
func RandomCurrency() string {
	currencies := []string{
		USD, USD, USD,
		EUR, EUR,
		EGP,
	}

	return currencies[rand.Intn(len(currencies))]
}


// RandomMoney generates a random amount of money in cents (smallest currency unit).
func RandomMoney() int64 {
    // Random amount between $10.00 and $50,000.00
    min := int64(1_000)      // $10.00 in cents
    max := int64(5_000_000)  // $50,000.00 in cents
    return min + rand.Int63n(max-min)
}

// RandomEmail generates a random email address using random first and last names, and a random number.
func RandomEmail() string {
	return fmt.Sprintf("%s@email.com", RandomOwner())
}

// RandomString generates a random string of the specified length using letters and digits.
func RandomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var sb strings.Builder
	for i := 0; i < n; i++ {
		sb.WriteByte(letters[rand.Intn(len(letters))])
	}
	return sb.String()
}