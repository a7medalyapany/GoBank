package db

import (
	"encoding/json"

	"github.com/a7medalyapany/GoBank.git/util"
)

// MarshalJSON customizes JSON serialization for Account
func (a Account) MarshalJSON() ([]byte, error) {
	type Alias Account // Prevent infinite recursion
	
	return json.Marshal(&struct {
		Balance float64 `json:"balance"` // Override balance field
		*Alias
	}{
		Balance: util.CentsToFloat(a.Balance),
		Alias:   (*Alias)(&a),
	})
}