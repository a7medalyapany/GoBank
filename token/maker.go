package token

import "time"

type Maker interface {

	// CreateToken creates a new token for a specific username and duration. It returns the token string and the payload data.
	CreateToken(username string, duration time.Duration) (string, *Payload, error)

	// VerifyToken checks if the token is valid or not. It returns the token payload if the token is valid.
	VerifyToken(token string) (*Payload, error)
}