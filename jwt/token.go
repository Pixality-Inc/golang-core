package jwt

import "time"

type Token interface {
	Claims() Claims
	IssuedAt() time.Time
	ExpiresAt() time.Time
	SetIssuedAt(date time.Time) Token
	SetExpiresAt(date time.Time) Token
}

type SignedToken interface {
	Token
	Signed() string
}

type TokenImpl struct {
	claims    Claims
	signed    string
	issuedAt  time.Time
	expiresAt time.Time
}

func NewToken(claims Claims) *TokenImpl {
	if claims == nil {
		claims = make(Claims)
	}

	return &TokenImpl{
		claims:    claims,
		signed:    "",
		issuedAt:  time.Now(),
		expiresAt: time.Now().Add(time.Hour),
	}
}

func newSignedToken(claims Claims, signed string) *TokenImpl {
	token := NewToken(claims)

	token.signed = signed

	return token
}

func (t *TokenImpl) Signed() string {
	return t.signed
}

func (t *TokenImpl) Claims() Claims {
	return t.claims
}

func (t *TokenImpl) IssuedAt() time.Time {
	return t.issuedAt
}

func (t *TokenImpl) ExpiresAt() time.Time {
	return t.expiresAt
}

func (t *TokenImpl) SetIssuedAt(date time.Time) Token {
	t.issuedAt = date

	return t
}

func (t *TokenImpl) SetExpiresAt(date time.Time) Token {
	t.expiresAt = date

	return t
}
