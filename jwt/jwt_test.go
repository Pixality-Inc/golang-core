package jwt

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestJwtEncodeDecode(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	iat := time.Now()
	eat := iat.Add(defaultTokenExpiration)

	jwtService := New("iddqd")

	token := NewToken(iat, nil)

	require.Equal(t, token.IssuedAt(), iat)
	require.Equal(t, token.ExpiresAt(), eat)
	require.Equal(t, jwt.MapClaims{}, token.Claims())

	signedToken, err := jwtService.Encode(ctx, token)
	require.NoError(t, err)

	require.Equal(t, token.IssuedAt(), signedToken.IssuedAt())
	require.Equal(t, token.ExpiresAt(), signedToken.ExpiresAt())
	require.Equal(t, token.Claims(), signedToken.Claims())

	decodedToken, err := jwtService.Decode(ctx, signedToken.Signed())
	require.NoError(t, err)

	require.Equal(t, token.IssuedAt().Unix(), decodedToken.IssuedAt().Unix())
	require.Equal(t, token.ExpiresAt().Unix(), decodedToken.ExpiresAt().Unix())
	require.InDelta(t, iat.Unix(), decodedToken.Claims()["iat"], 0.01)
	require.InDelta(t, eat.Unix(), decodedToken.Claims()["exp"], 0.01)
}

func TestJwtEncodeDecodeClaims(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	iat := time.Now()
	eat := iat.Add(defaultTokenExpiration)
	claims := jwt.MapClaims{
		"foo":   "bar",
		"hello": true,
	}

	jwtService := New("iddqd")

	token := NewToken(iat, claims)

	require.Equal(t, claims, token.Claims())

	signedToken, err := jwtService.Encode(ctx, token)
	require.NoError(t, err)

	require.Equal(t, claims["foo"], signedToken.Claims()["foo"])
	require.Equal(t, claims["hello"], signedToken.Claims()["hello"])
	require.InDelta(t, iat.Unix(), signedToken.Claims()["iat"], 0.01)
	require.InDelta(t, eat.Unix(), signedToken.Claims()["exp"], 0.01)

	decodedToken, err := jwtService.Decode(ctx, signedToken.Signed())
	require.NoError(t, err)

	require.Equal(t, claims["foo"], decodedToken.Claims()["foo"])
	require.Equal(t, claims["hello"], decodedToken.Claims()["hello"])
	require.InDelta(t, iat.Unix(), decodedToken.Claims()["iat"], 0.01)
	require.InDelta(t, eat.Unix(), decodedToken.Claims()["exp"], 0.01)
}

func TestJwtDecodeFailed(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	iat := time.Now()

	encoder := New("iddqd")
	decoder := New("iddqd1")

	token := NewToken(iat, nil)

	signedToken, err := encoder.Encode(ctx, token)
	require.NoError(t, err)

	_, err = decoder.Decode(ctx, signedToken.Signed())
	require.ErrorIs(t, err, jwt.ErrSignatureInvalid)
}

func TestJwtExpiredToken(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	iat := time.Now()
	eat := iat.Add(-24 * time.Hour)

	jwtService := New("iddqd")

	token := NewToken(iat, nil)

	token.SetExpiresAt(eat)

	signedToken, err := jwtService.Encode(ctx, token)
	require.NoError(t, err)

	_, err = jwtService.Decode(ctx, signedToken.Signed())
	require.ErrorIs(t, err, jwt.ErrTokenExpired)
}
