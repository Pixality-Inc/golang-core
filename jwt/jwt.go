package jwt

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/pixality-inc/golang-core/clock"
	"github.com/pixality-inc/golang-core/logger"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrSign                    = errors.New("sign failed")
	ErrUnexpectedSigningMethod = errors.New("unexpected signing method")
	ErrParse                   = errors.New("parse failed")
	ErrTokenIsInvalid          = errors.New("token is invalid")
	ErrExtractClaims           = errors.New("extract claims failed")
)

const defaultTokenExpiration = time.Hour

type Claims = jwt.MapClaims

type Jwt interface {
	Decode(ctx context.Context, signedString string) (Token, error)
	Encode(ctx context.Context, token Token) (SignedToken, error)
}

type Impl struct {
	log    logger.Loggable
	secret string
}

func New(secret string) *Impl {
	return &Impl{
		log:    logger.NewLoggableImplWithService("jwt"),
		secret: secret,
	}
}

func (j *Impl) Decode(ctx context.Context, signedString string) (Token, error) {
	clocks := clock.GetClock(ctx)

	jwtToken, err := jwt.Parse(signedString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: %v", ErrUnexpectedSigningMethod, token.Header["alg"])
		}

		return []byte(j.secret), nil
	})
	if err != nil {
		return nil, errors.Join(ErrParse, err)
	}

	if !jwtToken.Valid {
		return nil, ErrTokenIsInvalid
	}

	claims, ok := jwtToken.Claims.(Claims)
	if !ok {
		return nil, ErrExtractClaims
	}

	iat, err := claims.GetIssuedAt()
	if err != nil {
		iat = jwt.NewNumericDate(clocks.Now())
	}

	eat, err := claims.GetExpirationTime()
	if err != nil {
		eat = jwt.NewNumericDate(iat.Add(defaultTokenExpiration))
	}

	token := newSignedToken(iat.Time, claims, signedString)

	token.SetExpiresAt(eat.Time)

	return token, nil
}

func (j *Impl) Encode(_ context.Context, token Token) (SignedToken, error) {
	claims := token.Claims()

	if _, ok := claims["iat"]; !ok {
		claims["iat"] = token.IssuedAt().Unix()
	}

	if _, ok := claims["exp"]; !ok {
		claims["exp"] = token.ExpiresAt().Unix()
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedString, err := jwtToken.SignedString([]byte(j.secret))
	if err != nil {
		return nil, errors.Join(ErrSign, err)
	}

	signedToken := newSignedToken(token.IssuedAt(), claims, signedString)

	return signedToken, nil
}
