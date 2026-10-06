package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/Rede-Medica-D-Excelencia-Vitalis/vitalis-service-template/internal/adapter/http/middleware"
	"github.com/Rede-Medica-D-Excelencia-Vitalis/vitalis-service-template/internal/app"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Dependencies struct {
	ServiceName string
	Pool        *pgxpool.Pool
	Redis       *redis.Client
	Examples    *app.ExampleService
}

func NewRouter(deps Dependencies) http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.Recoverer)
	r.Use(chimw.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.InternalIdentity)

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "ok",
			"service": deps.ServiceName,
		})
	})

	r.Get("/ready", func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		if err := deps.Pool.Ping(ctx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{
				"status": "degraded",
				"error":  "postgres",
			})
			return
		}
		if deps.Redis != nil {
			if err := deps.Redis.Ping(ctx).Err(); err != nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]any{
					"status": "degraded",
					"error":  "redis",
				})
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	})

	r.Route("/v1", func(r chi.Router) {
		r.Post("/examples", func(w http.ResponseWriter, req *http.Request) {
			var body struct {
				Name string `json:"name"`
			}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				writeErr(w, http.StatusBadRequest, "VALIDATION_ERROR", "JSON inválido", middleware.RequestIDFromCtx(req.Context()))
				return
			}
			e, err := deps.Examples.Create(req.Context(), app.CreateExampleInput{
				Name:          body.Name,
				CorrelationID: middleware.RequestIDFromCtx(req.Context()),
			})
			if err != nil {
				writeErr(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), middleware.RequestIDFromCtx(req.Context()))
				return
			}
			writeJSON(w, http.StatusCreated, e)
		})
	})

	return r
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, code, msg, requestID string) {
	writeJSON(w, status, map[string]any{
		"erro":       msg,
		"codigo":     code,
		"request_id": requestID,
	})
}
