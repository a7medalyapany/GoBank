package token

import (
	"testing"
	"time"

	"github.com/a7medalyapany/GoBank.git/util"
	"github.com/stretchr/testify/require"
)

func TestPasetoMaker(t *testing.T) {
	maker, err := NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)

	username := util.RandomOwner()
	duration := time.Minute

	issuedAt := time.Now()
	expiredAt := issuedAt.Add(duration)

	token, payload, err := maker.CreateToken(username, duration)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NotNil(t, payload)

	require.NotZero(t, payload.ID)
	require.Equal(t, username, payload.Username)
	require.WithinDuration(t, issuedAt, payload.IssuedAt, time.Second)
	require.WithinDuration(t, expiredAt, payload.ExpiredAt, time.Second)

	verified, err := maker.VerifyToken(token)
	require.NoError(t, err)
	require.NotNil(t, verified)

	require.Equal(t, payload.ID, verified.ID)
	require.Equal(t, payload.Username, verified.Username)
	require.WithinDuration(t, payload.IssuedAt, verified.IssuedAt, time.Second)
	require.WithinDuration(t, payload.ExpiredAt, verified.ExpiredAt, time.Second)
}

func TestExpiredPasetoToken(t *testing.T) {
	maker, err := NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)

	token, payload, err := maker.CreateToken(util.RandomOwner(), -time.Minute)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NotNil(t, payload)

	verified, err := maker.VerifyToken(token)
	require.Error(t, err)
	require.EqualError(t, err, ErrExpiredToken.Error())
	require.Nil(t, verified)
}

func TestPasetoMakerWithInvalidKeySize(t *testing.T) {
	// chacha20poly1305 requires exactly 32 bytes — test both under and over
	shortKey := (util.RandomString(16))
	maker, err := NewPasetoMaker(shortKey)
	require.Error(t, err)
	require.Nil(t, maker)

	longKey := util.RandomString(64)
	maker, err = NewPasetoMaker(longKey)
	require.Error(t, err)
	require.Nil(t, maker)
}

func TestPasetoVerifyTokenWithTamperedToken(t *testing.T) {
	maker, err := NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)

	token, _, err := maker.CreateToken(util.RandomOwner(), time.Minute)
	require.NoError(t, err)

	// Tamper the token — PASETO's authenticated encryption will reject this
	tampered := token[:len(token)-4] + "xxxx"

	verified, err := maker.VerifyToken(tampered)
	require.Error(t, err)
	require.EqualError(t, err, ErrInvalidToken.Error())
	require.Nil(t, verified)
}

func TestPasetoVerifyTokenWithWrongKey(t *testing.T) {
	maker1, err := NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)

	maker2, err := NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)

	token, _, err := maker1.CreateToken(util.RandomOwner(), time.Minute)
	require.NoError(t, err)

	// maker2 has a different key — decryption must fail
	verified, err := maker2.VerifyToken(token)
	require.Error(t, err)
	require.EqualError(t, err, ErrInvalidToken.Error())
	require.Nil(t, verified)
}