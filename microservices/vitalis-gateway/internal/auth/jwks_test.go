package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func jwksServer(t *testing.T, key *rsa.PublicKey, kid string, hits *atomic.Int32, up *atomic.Bool) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		if !up.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kty": "RSA",
			"kid": kid,
			"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		}}})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestJWKS_CarregaECacheiaChave(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var hits atomic.Int32
	var up atomic.Bool
	up.Store(true)
	srv := jwksServer(t, &key.PublicKey, "k1", &hits, &up)

	j := NewJWKS(srv.URL, time.Hour)
	if j.Ready() {
		t.Fatal("Ready antes de buscar")
	}
	got, err := j.Key(context.Background(), "k1")
	if err != nil {
		t.Fatal(err)
	}
	if got.N.Cmp(key.N) != 0 || got.E != key.E {
		t.Fatal("chave reconstruída difere da original")
	}
	if !j.Ready() {
		t.Fatal("deveria estar Ready")
	}
	if _, err := j.Key(context.Background(), "k1"); err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 1 {
		t.Fatalf("buscou %d vezes, esperado 1 (cache)", hits.Load())
	}
}

func TestJWKS_KidDesconhecidoNaoViraFlood(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	var hits atomic.Int32
	var up atomic.Bool
	up.Store(true)
	srv := jwksServer(t, &key.PublicKey, "k1", &hits, &up)

	j := NewJWKS(srv.URL, time.Hour)
	for range 10 {
		if _, err := j.Key(context.Background(), "forjado"); err == nil {
			t.Fatal("kid desconhecido deveria falhar")
		}
	}
	if hits.Load() != 1 {
		t.Fatalf("buscou %d vezes, esperado 1 (minRefetch)", hits.Load())
	}
}

func TestJWKS_UsaCacheQuandoIdentityCai(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	var hits atomic.Int32
	var up atomic.Bool
	up.Store(true)
	srv := jwksServer(t, &key.PublicKey, "k1", &hits, &up)

	j := NewJWKS(srv.URL, time.Nanosecond) // cache "expira" na hora
	if _, err := j.Key(context.Background(), "k1"); err != nil {
		t.Fatal(err)
	}
	up.Store(false)
	if _, err := j.Key(context.Background(), "k1"); err != nil {
		t.Fatalf("deveria usar a chave em cache com o Identity fora: %v", err)
	}
}
