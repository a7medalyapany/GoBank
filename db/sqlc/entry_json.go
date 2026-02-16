package db

import (
	"encoding/json"

	"github.com/a7medalyapany/GoBank.git/util"
)

// MarshalJSON customizes JSON serialization for Entry
func (e Entry) MarshalJSON() ([]byte, error) {
	type Alias Entry
	
	return json.Marshal(&struct {
		Amount float64 `json:"amount"`
		*Alias
	}{
		Amount: util.CentsToFloat(e.Amount),
		Alias:  (*Alias)(&e),
	})
}