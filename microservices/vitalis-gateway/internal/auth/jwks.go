package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"
)

// Intervalo mínimo entre buscas quando chega um kid desconhecido,
// para um token forjado não virar flood de requisições no Identity.
const minRefetch = 30 * time.Second

// JWKS baixa e mantém em cache as chaves públicas do Identity, indexadas por kid.
type JWKS struct {
	url     string
	refresh time.Duration
	client  *http.Client

	mu      sync.RWMutex
	keys    map[string]*rsa.PublicKey
	checked time.Time // última tentativa de busca (com ou sem sucesso)
}

func NewJWKS(url string, refresh time.Duration) *JWKS {
	return &JWKS{
		url:     url,
		refresh: refresh,
		client:  &http.Client{Timeout: 5 * time.Second},
		keys:    map[string]*rsa.PublicKey{},
	}
}

// Key devolve a chave pública do kid, buscando de novo no Identity quando o cache expira.
func (j *JWKS) Key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	j.mu.RLock()
	key := j.keys[kid]
	age := time.Since(j.checked)
	j.mu.RUnlock()

	if key != nil && age < j.refresh {
		return key, nil
	}
	if key == nil && age < minRefetch {
		return nil, fmt.Errorf("kid %q desconhecido", kid)
	}

	if err := j.Refresh(ctx); err != nil {
		if key != nil {
			// Identity fora do ar: segue validando com a chave que já estava em cache.
			return key, nil
		}
		return nil, err
	}

	j.mu.RLock()
	defer j.mu.RUnlock()
	if key = j.keys[kid]; key == nil {
		return nil, fmt.Errorf("kid %q desconhecido", kid)
	}
	return key, nil
}

// Refresh busca o JWKS no Identity e substitui o cache inteiro.
func (j *JWKS) Refresh(ctx context.Context) error {
	j.mu.Lock()
	j.checked = time.Now()
	j.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, j.url, nil)
	if err != nil {
		return err
	}
	resp, err := j.client.Do(req)
	if err != nil {
		return fmt.Errorf("buscar JWKS: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS respondeu %d", resp.StatusCode)
	}

	var doc struct {
		Keys []struct {
			Kty string `json:"kty"`
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return fmt.Errorf("decodificar JWKS: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(doc.Keys))
	for _, k := range doc.Keys {
		if k.Kty != "RSA" || k.Kid == "" {
			continue
		}
		pub, err := rsaFromJWK(k.N, k.E)
		if err != nil {
			return fmt.Errorf("chave %s: %w", k.Kid, err)
		}
		keys[k.Kid] = pub
	}
	if len(keys) == 0 {
		return errors.New("JWKS sem chaves RSA")
	}

	j.mu.Lock()
	j.keys = keys
	j.mu.Unlock()
	return nil
}

// Ready indica se já existe ao menos uma chave carregada.
func (j *JWKS) Ready() bool {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return len(j.keys) > 0
}

// rsaFromJWK monta a chave a partir de n (módulo) e e (expoente), ambos base64url.
func rsaFromJWK(nB64, eB64 string) (*rsa.PublicKey, error) {
	nb, err := base64.RawURLEncoding.DecodeString(nB64)
	if err != nil {
		return nil, fmt.Errorf("n inválido: %w", err)
	}
	eb, err := base64.RawURLEncoding.DecodeString(eB64)
	if err != nil {
		return nil, fmt.Errorf("e inválido: %w", err)
	}
	e := 0
	for _, b := range eb {
		e = e<<8 | int(b)
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nb), E: e}, nil
}
