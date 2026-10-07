package auth

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrNotAccessToken = errors.New("token não é de acesso")

// Claims espelha o accessClaims emitido pelo Identity.
type Claims struct {
	UserID string   `json:"uid"`
	Email  string   `json:"email"`
	Roles  []string `json:"roles"`
	Type   string   `json:"typ"`
	jwt.RegisteredClaims
}

type Validator struct {
	jwks   *JWKS
	parser *jwt.Parser
}

func NewValidator(jwks *JWKS, issuer, audience string) *Validator {
	return &Validator{
		jwks: jwks,
		parser: jwt.NewParser(
			// Só RS256: bloqueia o ataque de trocar o alg para HS256 ou "none".
			jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
			jwt.WithIssuer(issuer),
			jwt.WithAudience(audience),
			jwt.WithExpirationRequired(),
			jwt.WithLeeway(30*time.Second),
		),
	}
}

// Validate confere assinatura (JWKS), alg, iss, aud, exp e o tipo do token.
func (v *Validator) Validate(ctx context.Context, raw string) (*Claims, error) {
	claims := &Claims{}
	_, err := v.parser.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		kid, _ := t.Header["kid"].(string)
		return v.jwks.Key(ctx, kid)
	})
	if err != nil {
		return nil, err
	}
	// Refresh token tem a mesma assinatura; sem essa checagem ele valeria como access por 7 dias.
	if claims.Type != "access" {
		return nil, ErrNotAccessToken
	}
	if claims.UserID == "" {
		return nil, errors.New("token sem uid")
	}
	return claims, nil
}
