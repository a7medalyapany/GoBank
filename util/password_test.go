package util

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// TestHashPassword tests the HashPassword and CheckPassword functions.
func TestPassword(t *testing.T) {
	password := RandomOwner()

	hashedPassword1, err := HashPassword(password)
	require.NoError(t, err)
	require.NotEmpty(t, hashedPassword1)
	

	err = CheckPassword(password, hashedPassword1)
	require.NoError(t, err)

	wrongPassword := "wrong-password"
	err = CheckPassword(wrongPassword, hashedPassword1)
	require.EqualError(t, err, bcrypt.ErrMismatchedHashAndPassword.Error())

	hashedPassword2, err := HashPassword(password)
	require.NoError(t, err)
	require.NotEmpty(t, hashedPassword2)
	// Ensure that two hashed passwords for the same password are different (due to salt)
	require.NotEqual(t, hashedPassword1, hashedPassword2)
}