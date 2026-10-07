package jwtadapter

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-identity/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

type fakeRefreshRepo struct{ saved map[string]string }

func (f *fakeRefreshRepo) Save(_ context.Context, jti, userID string, _ time.Time) error {
	f.saved[jti] = userID
	return nil
}
func (f *fakeRefreshRepo) FindValid(_ context.Context, jti string) (string, error) {
	return f.saved[jti], nil
}
func (f *fakeRefreshRepo) Revoke(_ context.Context, jti string) error {
	delete(f.saved, jti)
	return nil
}

func newTestIssuer(t *testing.T) (*Issuer, *fakeRefreshRepo) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	privPath := filepath.Join(dir, "priv.pem")
	pubPath := filepath.Join(dir, "pub.pem")

	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
	if err := os.WriteFile(privPath, privPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pubPath, pubPEM, 0o600); err != nil {
		t.Fatal(err)
	}

	repo := &fakeRefreshRepo{saved: map[string]string{}}
	iss, err := NewIssuer(privPath, pubPath, "vitalis-identity", "vitalis", 15, 7, repo)
	if err != nil {
		t.Fatal(err)
	}
	return iss, repo
}

func TestIssueAccess_TemKidETypAccess(t *testing.T) {
	iss, _ := newTestIssuer(t)
	user := domain.User{ID: "u1", Email: "ana@test.com", Roles: []string{"paciente"}}

	raw, expiresIn, err := iss.IssueAccess(user)
	if err != nil {
		t.Fatal(err)
	}
	if expiresIn != 15*60 {
		t.Errorf("expires_in = %d", expiresIn)
	}

	claims := &accessClaims{}
	tok, err := jwt.ParseWithClaims(raw, claims, func(*jwt.Token) (any, error) { return iss.public, nil },
		jwt.WithValidMethods([]string{"RS256"}), jwt.WithIssuer("vitalis-identity"), jwt.WithAudience("vitalis"))
	if err != nil {
		t.Fatal(err)
	}
	if tok.Header["kid"] != keyID {
		t.Errorf("kid = %v, esperado %s", tok.Header["kid"], keyID)
	}
	if claims.Type != "access" || claims.UserID != "u1" || len(claims.Roles) != 1 {
		t.Errorf("claims inesperados: %+v", claims)
	}
}

func TestRefresh_RoundTripEPersisteJTI(t *testing.T) {
	iss, repo := newTestIssuer(t)

	raw, err := iss.IssueRefresh(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	jti, userID, err := iss.ParseRefresh(raw)
	if err != nil {
		t.Fatal(err)
	}
	if userID != "u1" || repo.saved[jti] != "u1" {
		t.Fatalf("jti %q não persistido para u1 (repo: %v)", jti, repo.saved)
	}
}

func TestParseRefresh_RejeitaAccessToken(t *testing.T) {
	iss, _ := newTestIssuer(t)
	access, _, err := iss.IssueAccess(domain.User{ID: "u1", Roles: []string{"paciente"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := iss.ParseRefresh(access); err == nil {
		t.Fatal("access token não pode ser aceito como refresh")
	}
}

func TestParseRefresh_RejeitaOutraChave(t *testing.T) {
	iss, _ := newTestIssuer(t)
	other, _ := newTestIssuer(t)
	raw, err := other.IssueRefresh(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := iss.ParseRefresh(raw); err == nil {
		t.Fatal("refresh assinado por outra chave foi aceito")
	}
}

func TestPublicJWKS(t *testing.T) {
	iss, _ := newTestIssuer(t)
	doc, err := iss.PublicJWKS()
	if err != nil {
		t.Fatal(err)
	}
	keys := doc["keys"].([]map[string]any)
	if len(keys) != 1 || keys[0]["kid"] != keyID || keys[0]["alg"] != "RS256" {
		t.Fatalf("JWKS inesperado: %v", doc)
	}
	if _, ok := keys[0]["d"]; ok {
		t.Fatal("JWKS não pode expor a parte privada")
	}
}
