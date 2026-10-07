package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-gateway/internal/auth"
	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-gateway/internal/httpx"
	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-gateway/internal/middleware"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type Dependencies struct {
	ServiceName  string
	Log          *slog.Logger
	JWKS         *auth.JWKS
	Gateway      http.Handler
	CORSOrigins  []string
	RateLimiter  *middleware.RateLimiter
	MaxBodyBytes int64
}

func NewRouter(d Dependencies) http.Handler {
	r := chi.NewRouter()

	// Sem chimw.RealIP: o Gateway é a borda, e um X-Forwarded-For vindo do cliente
	// é forjável (permitiria burlar o rate limit por IP).
	r.Use(chimw.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logging(d.Log))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   d.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "Idempotency-Key", "X-Request-Id"},
		ExposedHeaders:   []string{"X-Request-Id"},
		AllowCredentials: true, // necessário para o cookie httpOnly do refresh (ADR-004)
		MaxAge:           300,
	}))
	r.Use(d.RateLimiter.Middleware)
	r.Use(maxBody(d.MaxBodyBytes))

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": d.ServiceName})
	})

	// Pronto = consegue validar JWT, ou seja, tem as chaves do Identity.
	r.Get("/ready", func(w http.ResponseWriter, req *http.Request) {
		if !d.JWKS.Ready() {
			ctx, cancel := context.WithTimeout(req.Context(), 3*time.Second)
			defer cancel()
			if err := d.JWKS.Refresh(ctx); err != nil {
				httpx.WriteJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "degraded", "error": "jwks"})
				return
			}
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	})

	// Todo o resto da API pública passa pelo handler do Gateway.
	r.Handle("/v1/*", d.Gateway)

	return r
}

func maxBody(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}
