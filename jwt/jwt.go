package jwt

import (
	"errors"
	"fmt"

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

type Claims = jwt.MapClaims

type Jwt interface {
	Decode(signedString string) (Token, error)
	Encode(token Token) (SignedToken, error)
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

func (j *Impl) Decode(signedString string) (Token, error) {
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

	token := newSignedToken(claims, signedString)

	return token, nil
}

func (j *Impl) Encode(token Token) (SignedToken, error) {
	claims := token.Claims()

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	jwtToken.Header["iat"] = token.IssuedAt().Unix()
	jwtToken.Header["exp"] = token.ExpiresAt().Unix()

	signedString, err := jwtToken.SignedString([]byte(j.secret))
	if err != nil {
		return nil, errors.Join(ErrSign, err)
	}

	signedToken := newSignedToken(claims, signedString)

	return signedToken, nil
}
