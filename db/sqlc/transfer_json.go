package db

import (
	"encoding/json"

	"github.com/a7medalyapany/GoBank.git/util"
)

// MarshalJSON customizes JSON serialization for Transfer
func (t Transfer) MarshalJSON() ([]byte, error) {
	type Alias Transfer
	
	return json.Marshal(&struct {
		Amount float64 `json:"amount"`
		*Alias
	}{
		Amount: util.CentsToFloat(t.Amount),
		Alias:  (*Alias)(&t),
	})
}