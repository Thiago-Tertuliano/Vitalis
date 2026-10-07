package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var ok = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

func TestRateLimiter_BloqueiaAposBurst(t *testing.T) {
	h := NewRateLimiter(1, 2).Middleware(ok)

	codes := make([]int, 0, 3)
	for range 3 {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		codes = append(codes, rec.Code)
	}
	if codes[0] != 200 || codes[1] != 200 || codes[2] != http.StatusTooManyRequests {
		t.Fatalf("códigos = %v, esperado [200 200 429]", codes)
	}

	// Outro IP tem o próprio balde.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.2:1234"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("outro IP recebeu %d", rec.Code)
	}
}

func TestRateLimiter_IgnoraXForwardedFor(t *testing.T) {
	h := NewRateLimiter(1, 1).Middleware(ok)
	for i, xff := range []string{"1.1.1.1", "2.2.2.2"} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.3:1234"
		req.Header.Set("X-Forwarded-For", xff)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if i == 1 && rec.Code != http.StatusTooManyRequests {
			t.Fatalf("X-Forwarded-For burlou o rate limit (status %d)", rec.Code)
		}
	}
}

func TestRequestID(t *testing.T) {
	var seen string
	h := RequestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = r.Header.Get("X-Request-Id")
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if seen == "" || rec.Header().Get("X-Request-Id") != seen {
		t.Fatalf("id gerado %q, devolvido %q", seen, rec.Header().Get("X-Request-Id"))
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-Id", "abc-123")
	h.ServeHTTP(httptest.NewRecorder(), req)
	if seen != "abc-123" {
		t.Fatalf("id do cliente não foi propagado: %q", seen)
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-Id", strings.Repeat("a", 500))
	h.ServeHTTP(httptest.NewRecorder(), req)
	if len(seen) > 128 {
		t.Fatalf("id gigante não foi substituído (len %d)", len(seen))
	}
}
