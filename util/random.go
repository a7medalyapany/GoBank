package util

import (
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

func RandomOwner() string {
	first := firstNames[rand.Intn(len(firstNames))]
	last := lastNames[rand.Intn(len(lastNames))]
	number := rand.Intn(1000)

	// e.g. ahmed.alyapany69
	return strings.ToLower(
		first + "." + last + string(rune('0'+number%10)),
	)
}


func RandomCurrency() string {
	currencies := []string{
		USD, USD, USD,
		EUR, EUR,
		EGP,
	}

	return currencies[rand.Intn(len(currencies))]
}


func RandomMoney() int64 {
    // Random amount between $10.00 and $50,000.00
    min := int64(1_000)      // $10.00 in cents
    max := int64(5_000_000)  // $50,000.00 in cents
    return min + rand.Int63n(max-min)
}