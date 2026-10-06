package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-identity/internal/adapter/http/middleware"
	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-identity/internal/app"
	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-identity/internal/domain"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Dependencies struct {
	ServiceName string
	Pool        *pgxpool.Pool
	Redis       *redis.Client
	Auth        *app.AuthService
	Profile     *app.ProfileService
	Addresses   *app.AddressService
	Admin       *app.AdminService
	Passwords   *app.PasswordResetService
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
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "degraded", "error": "postgres"})
			return
		}
		if deps.Redis != nil {
			if err := deps.Redis.Ping(ctx).Err(); err != nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "degraded", "error": "redis"})
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	})

	rid := func(req *http.Request) string { return middleware.RequestIDFromCtx(req.Context()) }

	r.Route("/v1", func(r chi.Router) {
		// ---- AUTH (público) ----
		r.Post("/auth/registro", func(w http.ResponseWriter, req *http.Request) {
			var body struct {
				Nome           string `json:"nome"`
				Email          string `json:"email"`
				Senha          string `json:"senha"`
				Telefone       string `json:"telefone"`
				TipoUsuario    string `json:"tipo_usuario"`
				CPF            string `json:"cpf"`
				DataNascimento string `json:"data_nascimento"`
			}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				writeErr(w, 400, "VALIDATION_ERROR", "JSON inválido", rid(req))
				return
			}
			var dn *time.Time
			if body.DataNascimento != "" {
				t, err := time.Parse("2006-01-02", body.DataNascimento)
				if err != nil {
					writeErr(w, 400, "VALIDATION_ERROR", "data_nascimento inválida", rid(req))
					return
				}
				dn = &t
			}
			out, err := deps.Auth.Register(req.Context(), app.RegisterInput{
				Nome: body.Nome, Email: body.Email, Senha: body.Senha, Telefone: body.Telefone,
				TipoUsuario: body.TipoUsuario, CPF: body.CPF, DataNascimento: dn,
				CorrelationID: rid(req),
			})
			if errors.Is(err, app.ErrEmailJaExiste) {
				writeErr(w, 409, "CONFLICT", err.Error(), rid(req))
				return
			}
			if errors.Is(err, app.ErrRoleInvalida) {
				writeErr(w, 400, "VALIDATION_ERROR", err.Error(), rid(req))
				return
			}
			if err != nil {
				writeErr(w, 400, "VALIDATION_ERROR", err.Error(), rid(req))
				return
			}
			writeJSON(w, 201, out)
		})

		r.Post("/auth/login", func(w http.ResponseWriter, req *http.Request) {
			var body struct {
				Email string `json:"email"`
				Senha string `json:"senha"`
			}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				writeErr(w, 400, "VALIDATION_ERROR", "JSON inválido", rid(req))
				return
			}
			out, err := deps.Auth.Login(req.Context(), app.LoginInput{Email: body.Email, Senha: body.Senha})
			if errors.Is(err, app.ErrCredenciaisInvalidas) || errors.Is(err, app.ErrUsuarioInativo) {
				writeErr(w, 401, "UNAUTHORIZED", err.Error(), rid(req))
				return
			}
			if err != nil {
				writeErr(w, 500, "INTERNAL", "erro interno", rid(req))
				return
			}
			writeJSON(w, 200, out)
		})

		r.Post("/auth/refresh", func(w http.ResponseWriter, req *http.Request) {
			var body struct {
				RefreshToken string `json:"refresh_token"`
			}
			_ = json.NewDecoder(req.Body).Decode(&body)
			out, err := deps.Auth.Refresh(req.Context(), app.RefreshInput{RefreshToken: body.RefreshToken})
			if errors.Is(err, app.ErrRefreshInvalido) || errors.Is(err, app.ErrUsuarioInativo) {
				writeErr(w, 401, "UNAUTHORIZED", err.Error(), rid(req))
				return
			}
			if err != nil {
				writeErr(w, 500, "INTERNAL", "erro interno", rid(req))
				return
			}
			writeJSON(w, 200, out)
		})

		r.Post("/auth/logout", func(w http.ResponseWriter, req *http.Request) {
			var body struct {
				RefreshToken string `json:"refresh_token"`
			}
			_ = json.NewDecoder(req.Body).Decode(&body)
			_ = deps.Auth.Logout(req.Context(), body.RefreshToken)
			w.WriteHeader(http.StatusNoContent)
		})

		r.Get("/auth/jwks.json", func(w http.ResponseWriter, req *http.Request) {
			jwks, err := deps.Auth.JWKS()
			if err != nil {
				writeErr(w, 500, "INTERNAL", "jwks", rid(req))
				return
			}
			writeJSON(w, 200, jwks)
		})

		r.Post("/auth/esqueci-senha", func(w http.ResponseWriter, req *http.Request) {
			var body struct {
				Email string `json:"email"`
			}
			_ = json.NewDecoder(req.Body).Decode(&body)
			token, err := deps.Passwords.Forgot(req.Context(), body.Email)
			if err != nil {
				writeErr(w, 500, "INTERNAL", "erro interno", rid(req))
				return
			}
			// MVP local: devolve token para testar na live (em prod só 202)
			resp := map[string]any{"status": "accepted"}
			if token != "" {
				resp["reset_token_dev"] = token
			}
			writeJSON(w, 202, resp)
		})

		r.Post("/auth/redefinir-senha", func(w http.ResponseWriter, req *http.Request) {
			var body struct {
				Token    string `json:"token"`
				NovaSenha string `json:"nova_senha"`
			}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				writeErr(w, 400, "VALIDATION_ERROR", "JSON inválido", rid(req))
				return
			}
			err := deps.Passwords.Redefinir(req.Context(), body.Token, body.NovaSenha)
			if errors.Is(err, app.ErrResetInvalido) {
				writeErr(w, 400, "VALIDATION_ERROR", err.Error(), rid(req))
				return
			}
			if err != nil {
				writeErr(w, 400, "VALIDATION_ERROR", err.Error(), rid(req))
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})

		// ---- USUÁRIO (headers Gateway) ----
		r.Get("/usuarios/me", func(w http.ResponseWriter, req *http.Request) {
			uid := middleware.UserIDFromCtx(req.Context())
			u, err := deps.Profile.Me(req.Context(), uid)
			if errors.Is(err, app.ErrNaoAutenticado) {
				writeErr(w, 401, "UNAUTHORIZED", err.Error(), rid(req))
				return
			}
			if err != nil {
				writeErr(w, 404, "NOT_FOUND", "usuário não encontrado", rid(req))
				return
			}
			writeJSON(w, 200, map[string]any{
				"id": u.ID, "nome": u.Nome, "email": u.Email, "telefone": u.Telefone,
				"status": u.Status, "roles": u.Roles, "cpf": u.CPF,
			})
		})

		r.Put("/usuarios/me", func(w http.ResponseWriter, req *http.Request) {
			uid := middleware.UserIDFromCtx(req.Context())
			var body struct {
				Nome           string `json:"nome"`
				Telefone       string `json:"telefone"`
				CPF            string `json:"cpf"`
				DataNascimento string `json:"data_nascimento"`
			}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				writeErr(w, 400, "VALIDATION_ERROR", "JSON inválido", rid(req))
				return
			}
			var dn *time.Time
			if body.DataNascimento != "" {
				t, err := time.Parse("2006-01-02", body.DataNascimento)
				if err != nil {
					writeErr(w, 400, "VALIDATION_ERROR", "data_nascimento inválida", rid(req))
					return
				}
				dn = &t
			}
			u, err := deps.Profile.Update(req.Context(), uid, app.UpdateProfileInput{
				Nome: body.Nome, Telefone: body.Telefone, CPF: body.CPF, DataNascimento: dn,
				CorrelationID: rid(req),
			})
			if errors.Is(err, app.ErrNaoAutenticado) {
				writeErr(w, 401, "UNAUTHORIZED", err.Error(), rid(req))
				return
			}
			if err != nil {
				writeErr(w, 400, "VALIDATION_ERROR", err.Error(), rid(req))
				return
			}
			writeJSON(w, 200, map[string]any{
				"id": u.ID, "nome": u.Nome, "email": u.Email, "telefone": u.Telefone, "roles": u.Roles,
			})
		})

		r.Get("/usuarios/me/enderecos", func(w http.ResponseWriter, req *http.Request) {
			uid := middleware.UserIDFromCtx(req.Context())
			list, err := deps.Addresses.List(req.Context(), uid)
			if errors.Is(err, app.ErrNaoAutenticado) {
				writeErr(w, 401, "UNAUTHORIZED", err.Error(), rid(req))
				return
			}
			if err != nil {
				writeErr(w, 500, "INTERNAL", "erro interno", rid(req))
				return
			}
			writeJSON(w, 200, list)
		})

		r.Post("/usuarios/me/enderecos", func(w http.ResponseWriter, req *http.Request) {
			uid := middleware.UserIDFromCtx(req.Context())
			var body domain.Address
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				writeErr(w, 400, "VALIDATION_ERROR", "JSON inválido", rid(req))
				return
			}
			a, err := deps.Addresses.Create(req.Context(), uid, body)
			if errors.Is(err, app.ErrNaoAutenticado) {
				writeErr(w, 401, "UNAUTHORIZED", err.Error(), rid(req))
				return
			}
			if err != nil {
				writeErr(w, 400, "VALIDATION_ERROR", err.Error(), rid(req))
				return
			}
			writeJSON(w, 201, a)
		})

		r.Put("/usuarios/me/enderecos/{id}", func(w http.ResponseWriter, req *http.Request) {
			uid := middleware.UserIDFromCtx(req.Context())
			var body domain.Address
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				writeErr(w, 400, "VALIDATION_ERROR", "JSON inválido", rid(req))
				return
			}
			body.ID = chi.URLParam(req, "id")
			if err := deps.Addresses.Update(req.Context(), uid, body); err != nil {
				if errors.Is(err, app.ErrNaoAutenticado) {
					writeErr(w, 401, "UNAUTHORIZED", err.Error(), rid(req))
					return
				}
				writeErr(w, 400, "VALIDATION_ERROR", err.Error(), rid(req))
				return
			}
			writeJSON(w, 200, map[string]string{"status": "ok"})
		})

		r.Delete("/usuarios/me/enderecos/{id}", func(w http.ResponseWriter, req *http.Request) {
			uid := middleware.UserIDFromCtx(req.Context())
			id := chi.URLParam(req, "id")
			if err := deps.Addresses.Delete(req.Context(), uid, id); err != nil {
				if errors.Is(err, app.ErrNaoAutenticado) {
					writeErr(w, 401, "UNAUTHORIZED", err.Error(), rid(req))
					return
				}
				writeErr(w, 400, "VALIDATION_ERROR", err.Error(), rid(req))
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})

		r.Put("/admin/usuarios/{id}/roles", func(w http.ResponseWriter, req *http.Request) {
			var body struct {
				Roles []string `json:"roles"`
			}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				writeErr(w, 400, "VALIDATION_ERROR", "JSON inválido", rid(req))
				return
			}
			err := deps.Admin.ReplaceRoles(
				req.Context(),
				middleware.RolesFromCtx(req.Context()),
				chi.URLParam(req, "id"),
				body.Roles,
				rid(req),
			)
			if errors.Is(err, app.ErrForbidden) {
				writeErr(w, 403, "FORBIDDEN", err.Error(), rid(req))
				return
			}
			if err != nil {
				writeErr(w, 400, "VALIDATION_ERROR", err.Error(), rid(req))
				return
			}
			writeJSON(w, 200, map[string]any{"roles": body.Roles})
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