package gateway

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-gateway/internal/auth"
	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-gateway/internal/proxy"
	"github.com/golang-jwt/jwt/v5"
)

const (
	testKID      = "test-kid"
	testIssuer   = "vitalis-identity"
	testAudience = "vitalis"
)

var (
	keyOnce  sync.Once
	signKey  *rsa.PrivateKey
	otherKey *rsa.PrivateKey
)

func keys(t *testing.T) (*rsa.PrivateKey, *rsa.PrivateKey) {
	t.Helper()
	keyOnce.Do(func() {
		var err error
		if signKey, err = rsa.GenerateKey(rand.Reader, 2048); err != nil {
			t.Fatal(err)
		}
		if otherKey, err = rsa.GenerateKey(rand.Reader, 2048); err != nil {
			t.Fatal(err)
		}
	})
	return signKey, otherKey
}

// upstreamEcho devolve o que o serviço de destino recebeu do Gateway.
type upstreamEcho struct {
	UserID  string `json:"user_id"`
	Roles   string `json:"roles"`
	Gateway string `json:"gateway"`
	Query   string `json:"query"`
}

type env struct {
	handler http.Handler
	key     *rsa.PrivateKey
	other   *rsa.PrivateKey
}

func newEnv(t *testing.T) *env {
	t.Helper()
	key, other := keys(t)

	jwksSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kty": "RSA",
			"kid": testKID,
			"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		}}})
	}))
	t.Cleanup(jwksSrv.Close)

	echo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(upstreamEcho{
			UserID:  r.Header.Get("X-User-Id"),
			Roles:   r.Header.Get("X-Roles"),
			Gateway: r.Header.Get("X-Gateway"),
			Query:   r.URL.RawQuery,
		})
	}))
	t.Cleanup(echo.Close)

	// Porta fechada simula um serviço fora do ar.
	down := httptest.NewServer(http.NotFoundHandler())
	downURL := down.URL
	down.Close()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	reg, err := proxy.NewRegistry(map[string]string{
		"identity": echo.URL,
		"audit":    echo.URL,
		"comms":    echo.URL,
		"orders":   downURL,
	}, 5*time.Second, log)
	if err != nil {
		t.Fatal(err)
	}

	jwks := auth.NewJWKS(jwksSrv.URL, time.Minute)
	v := auth.NewValidator(jwks, testIssuer, testAudience)
	return &env{handler: NewHandler(v, reg, log), key: key, other: other}
}

type tokenOpts struct {
	typ    string
	roles  []string
	issuer string
	exp    time.Duration
	key    *rsa.PrivateKey
	method jwt.SigningMethod
}

func (e *env) token(t *testing.T, o tokenOpts) string {
	t.Helper()
	if o.typ == "" {
		o.typ = "access"
	}
	if o.issuer == "" {
		o.issuer = testIssuer
	}
	if o.exp == 0 {
		o.exp = 15 * time.Minute
	}
	if o.key == nil {
		o.key = e.key
	}
	if o.method == nil {
		o.method = jwt.SigningMethodRS256
	}
	now := time.Now()
	claims := auth.Claims{
		UserID: "user-123",
		Email:  "ana@test.com",
		Roles:  o.roles,
		Type:   o.typ,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    o.issuer,
			Subject:   "user-123",
			Audience:  []string{testAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(o.exp)),
		},
	}
	tk := jwt.NewWithClaims(o.method, claims)
	tk.Header["kid"] = testKID
	var signed string
	var err error
	if o.method == jwt.SigningMethodHS256 {
		// Ataque clássico: assinar HMAC usando a chave pública como segredo.
		signed, err = tk.SignedString(e.key.N.Bytes())
	} else {
		signed, err = tk.SignedString(o.key)
	}
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

func (e *env) do(method, target, token string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	return rec
}

func decodeEcho(t *testing.T, rec *httptest.ResponseRecorder) upstreamEcho {
	t.Helper()
	var got upstreamEcho
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("resposta do upstream inválida: %v (%s)", err, rec.Body.String())
	}
	return got
}

func TestHandler_StatusCodes(t *testing.T) {
	e := newEnv(t)
	paciente := e.token(t, tokenOpts{roles: []string{"paciente"}})
	admin := e.token(t, tokenOpts{roles: []string{"admin"}})

	cases := []struct {
		name    string
		method  string
		path    string
		token   string
		headers map[string]string
		want    int
	}{
		{"rota pública sem token", http.MethodPost, "/v1/auth/login", "", nil, http.StatusOK},
		{"token válido", http.MethodGet, "/v1/usuarios/me", paciente, nil, http.StatusOK},
		{"sem token", http.MethodGet, "/v1/usuarios/me", "", nil, http.StatusUnauthorized},
		{"X-User-Id forjado sem token", http.MethodGet, "/v1/usuarios/me", "", map[string]string{"X-User-Id": "outro"}, http.StatusUnauthorized},
		{"refresh usado como access", http.MethodGet, "/v1/usuarios/me", e.token(t, tokenOpts{typ: "refresh"}), nil, http.StatusUnauthorized},
		{"issuer errado", http.MethodGet, "/v1/usuarios/me", e.token(t, tokenOpts{issuer: "outro"}), nil, http.StatusUnauthorized},
		{"token expirado", http.MethodGet, "/v1/usuarios/me", e.token(t, tokenOpts{exp: -time.Hour}), nil, http.StatusUnauthorized},
		{"assinado com outra chave", http.MethodGet, "/v1/usuarios/me", e.token(t, tokenOpts{key: e.other}), nil, http.StatusUnauthorized},
		{"alg trocado para HS256", http.MethodGet, "/v1/usuarios/me", e.token(t, tokenOpts{method: jwt.SigningMethodHS256}), nil, http.StatusUnauthorized},
		{"token malformado", http.MethodGet, "/v1/usuarios/me", "abc.def.ghi", nil, http.StatusUnauthorized},
		{"paciente em rota admin", http.MethodPut, "/v1/admin/usuarios/x/roles", paciente, nil, http.StatusForbidden},
		{"paciente em audit", http.MethodGet, "/v1/audit/eventos", paciente, nil, http.StatusForbidden},
		{"admin em audit", http.MethodGet, "/v1/audit/eventos", admin, nil, http.StatusOK},
		{"rota desconhecida", http.MethodGet, "/v1/naoexiste", paciente, nil, http.StatusNotFound},
		{"rota interna não exposta", http.MethodGet, "/v1/internal/users", paciente, nil, http.StatusNotFound},
		{"prefixo parecido não casa", http.MethodGet, "/v1/usuariosX", paciente, nil, http.StatusNotFound},
		{"path traversal", http.MethodGet, "/v1/auth/../admin/usuarios", "", nil, http.StatusBadRequest},
		{"serviço não configurado", http.MethodPost, "/v1/webhooks/pagamentos", "", nil, http.StatusServiceUnavailable},
		{"serviço fora do ar", http.MethodGet, "/v1/pedidos", paciente, nil, http.StatusBadGateway},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := e.do(tc.method, tc.path, tc.token, tc.headers)
			if rec.Code != tc.want {
				t.Fatalf("status = %d, esperado %d (body: %s)", rec.Code, tc.want, rec.Body.String())
			}
		})
	}
}

func TestHandler_InjetaIdentidadeConfiavel(t *testing.T) {
	e := newEnv(t)
	tk := e.token(t, tokenOpts{roles: []string{"paciente", "medico"}})

	rec := e.do(http.MethodGet, "/v1/usuarios/me", tk, map[string]string{
		"X-User-Id": "atacante",
		"X-Roles":   "admin",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	got := decodeEcho(t, rec)
	if got.UserID != "user-123" {
		t.Errorf("X-User-Id = %q, esperado o uid do token", got.UserID)
	}
	if got.Roles != "paciente,medico" {
		t.Errorf("X-Roles = %q, esperado as roles do token", got.Roles)
	}
	if got.Gateway != "1" {
		t.Errorf("X-Gateway = %q", got.Gateway)
	}
}

func TestHandler_RotaPublicaNaoRepassaIdentidadeForjada(t *testing.T) {
	e := newEnv(t)
	rec := e.do(http.MethodPost, "/v1/auth/login", "", map[string]string{"X-User-Id": "atacante"})
	got := decodeEcho(t, rec)
	if got.UserID != "" {
		t.Errorf("X-User-Id = %q, esperado vazio em rota pública", got.UserID)
	}
}

func TestHandler_WebSocketTokenPorQueryNaoVazaParaUpstream(t *testing.T) {
	e := newEnv(t)
	tk := e.token(t, tokenOpts{roles: []string{"paciente"}})

	rec := e.do(http.MethodGet, "/v1/realtime?sala=1&access_token="+tk, "", map[string]string{
		"Connection": "Upgrade",
		"Upgrade":    "websocket",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (body: %s)", rec.Code, rec.Body.String())
	}
	got := decodeEcho(t, rec)
	if strings.Contains(got.Query, "access_token") {
		t.Errorf("query repassada contém o token: %q", got.Query)
	}
	if got.Query != "sala=1" {
		t.Errorf("query = %q, esperado sala=1", got.Query)
	}
}

func TestHandler_TokenPorQuerySoValeParaWebSocket(t *testing.T) {
	e := newEnv(t)
	tk := e.token(t, tokenOpts{roles: []string{"paciente"}})
	rec := e.do(http.MethodGet, "/v1/usuarios/me?access_token="+tk, "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, esperado 401", rec.Code)
	}
}
