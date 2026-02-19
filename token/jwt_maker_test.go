package token

import (
	"testing"
	"time"

	"github.com/a7medalyapany/GoBank.git/util"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestJWTMaker(t *testing.T) {
	maker, err := NewJWTMaker(util.RandomString(32))
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

func TestExpiredJWTToken(t *testing.T) {
	maker, err := NewJWTMaker(util.RandomString(32))
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

func TestInvalidJWTTokenAlgNone(t *testing.T) {
	payload, err := NewPayload(util.RandomOwner(), time.Minute)
	require.NoError(t, err)

	// Manually craft a token with alg:none to simulate an attack
	claims := &jwtClaims{
		ID:       payload.ID,
		Username: payload.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        payload.ID.String(),
			Subject:   payload.Username,
			IssuedAt:  jwt.NewNumericDate(payload.IssuedAt),
			ExpiresAt: jwt.NewNumericDate(payload.ExpiredAt),
		},
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	token, err := jwtToken.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	maker, err := NewJWTMaker(util.RandomString(32))
	require.NoError(t, err)

	verified, err := maker.VerifyToken(token)
	require.Error(t, err)
	require.EqualError(t, err, ErrInvalidToken.Error())
	require.Nil(t, verified)
}

func TestJWTMakerWithShortKey(t *testing.T) {
	maker, err := NewJWTMaker(util.RandomString(minSecretKeySize - 1))
	require.Error(t, err)
	require.Nil(t, maker)
}

func TestJWTVerifyTokenWithTamperedPayload(t *testing.T) {
	maker, err := NewJWTMaker(util.RandomString(32))
	require.NoError(t, err)

	token, _, err := maker.CreateToken(util.RandomOwner(), time.Minute)
	require.NoError(t, err)

	// Tamper the token by flipping a character in the signature segment
	tampered := token[:len(token)-4] + "xxxx"

	verified, err := maker.VerifyToken(tampered)
	require.Error(t, err)
	require.EqualError(t, err, ErrInvalidToken.Error())
	require.Nil(t, verified)
}