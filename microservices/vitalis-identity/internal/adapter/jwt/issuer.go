package jwtadapter

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-identity/internal/domain"
	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-identity/internal/port"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Issuer assina access/refresh RS256 e expõe JWKS (chave pública).
// Live: privada só no Identity; Gateway valida via JWKS.
type Issuer struct {
	private       *rsa.PrivateKey
	public        *rsa.PublicKey
	issuer        string
	audience      string
	accessTTL     time.Duration
	refreshTTL    time.Duration
	refreshTokens port.RefreshTokenRepository
}

type accessClaims struct {
	UserID string   `json:"uid"`
	Email  string   `json:"email"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}

type refreshClaims struct {
	UserID string `json:"uid"`
	jwt.RegisteredClaims
}

func NewIssuer(
	privateKeyPath, publicKeyPath, issuer, audience string,
	accessTTLMin, refreshTTLDays int,
	refreshTokens port.RefreshTokenRepository,
) (*Issuer, error) {
	priv, err := loadPrivateKey(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("private key: %w", err)
	}
	pub, err := loadPublicKey(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("public key: %w", err)
	}
	return &Issuer{
		private:       priv,
		public:        pub,
		issuer:        issuer,
		audience:      audience,
		accessTTL:     time.Duration(accessTTLMin) * time.Minute,
		refreshTTL:    time.Duration(refreshTTLDays) * 24 * time.Hour,
		refreshTokens: refreshTokens,
	}, nil
}

func (i *Issuer) IssueAccess(user domain.User) (string, int, error) {
	now := time.Now().UTC()
	exp := now.Add(i.accessTTL)
	claims := accessClaims{
		UserID: user.ID,
		Email:  user.Email,
		Roles:  user.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    i.issuer,
			Subject:   user.ID,
			Audience:  []string{i.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
			ID:        uuid.NewString(),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := t.SignedString(i.private)
	if err != nil {
		return "", 0, err
	}
	return signed, int(i.accessTTL.Seconds()), nil
}

func (i *Issuer) IssueRefresh(ctx context.Context, userID string) (string, error) {
	now := time.Now().UTC()
	exp := now.Add(i.refreshTTL)
	jti := uuid.NewString()

	if err := i.refreshTokens.Save(ctx, jti, userID, exp); err != nil {
		return "", fmt.Errorf("persist refresh jti: %w", err)
	}

	claims := refreshClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    i.issuer,
			Subject:   userID,
			Audience:  []string{i.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
			ID:        jti,
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return t.SignedString(i.private)
}

func (i *Issuer) ParseRefresh(token string) (jti, userID string, err error) {
	parsed, err := jwt.ParseWithClaims(token, &refreshClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodRS256.Alg() {
			return nil, fmt.Errorf("alg inválido")
		}
		return i.public, nil
	})
	if err != nil {
		return "", "", err
	}
	claims, ok := parsed.Claims.(*refreshClaims)
	if !ok || !parsed.Valid {
		return "", "", fmt.Errorf("refresh inválido")
	}
	return claims.ID, claims.UserID, nil
}

func (i *Issuer) PublicJWKS() (map[string]any, error) {
	n := i.public.N
	e := big.NewInt(int64(i.public.E))
	return map[string]any{
		"keys": []map[string]any{
			{
				"kty": "RSA",
				"use": "sig",
				"alg": "RS256",
				"kid": "vitalis-identity-1",
				"n":   base64.RawURLEncoding.EncodeToString(n.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(e.Bytes()),
			},
		},
	}, nil
}

func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, fmt.Errorf("PEM privado inválido")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// tenta PKCS8
		k, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, err
		}
		rk, ok := k.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("não é RSA")
		}
		return rk, nil
	}
	return key, nil
}

func loadPublicKey(path string) (*rsa.PublicKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, fmt.Errorf("PEM público inválido")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rk, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("não é RSA")
	}
	return rk, nil
}

// Garante na compilação que Issuer implementa a porta.
var _ port.TokenIssuer = (*Issuer)(nil)