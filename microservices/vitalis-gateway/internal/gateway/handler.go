package gateway

import (
	"log/slog"
	"net/http"
	"path"
	"strings"

	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-gateway/internal/auth"
	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-gateway/internal/httpx"
	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-gateway/internal/proxy"
	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-gateway/internal/routes"
)

// Headers que só o Gateway pode definir; qualquer valor vindo do cliente é descartado.
var internalHeaders = []string{"X-User-Id", "X-Roles", "X-Gateway"}

// Handler é o coração do Gateway: rota -> autenticação -> autorização -> proxy.
type Handler struct {
	validator *auth.Validator
	proxies   *proxy.Registry
	log       *slog.Logger
}

func NewHandler(v *auth.Validator, p *proxy.Registry, log *slog.Logger) *Handler {
	return &Handler{validator: v, proxies: p, log: log}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1) Anti-spoofing: o cliente não pode se passar por outro usuário mandando X-User-Id.
	for _, k := range internalHeaders {
		r.Header.Del(k)
	}

	// 2) Bloqueia "/v1/auth/../admin/..." (escaparia da regra da rota pública).
	if !isCleanPath(r.URL.Path) {
		httpx.WriteError(w, r, http.StatusBadRequest, "INVALID_PATH", "caminho inválido")
		return
	}

	// 3) Descobre para qual serviço vai.
	route, ok := routes.Match(r.URL.Path)
	if !ok {
		httpx.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", "rota não encontrada")
		return
	}

	// 4) Autenticação + autorização grossa (roles). Ownership fica no serviço (ADR-004).
	if route.Access == routes.Authenticated {
		raw, fromQuery := bearerToken(r)
		if raw == "" {
			httpx.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "token ausente")
			return
		}
		claims, err := h.validator.Validate(r.Context(), raw)
		if err != nil {
			h.log.Debug("jwt rejeitado", "err", err, "request_id", r.Header.Get("X-Request-Id"))
			httpx.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "token inválido ou expirado")
			return
		}
		if !hasAnyRole(claims.Roles, route.Roles) {
			httpx.WriteError(w, r, http.StatusForbidden, "FORBIDDEN", "sem permissão para esta rota")
			return
		}
		if fromQuery {
			stripQueryToken(r)
		}
		// 5) Injeta a identidade confiável para o serviço de destino.
		r.Header.Set("X-User-Id", claims.UserID)
		r.Header.Set("X-Roles", strings.Join(claims.Roles, ","))
	}
	r.Header.Set("X-Gateway", "1")

	// 6) Encaminha.
	rp, ok := h.proxies.Get(route.Upstream)
	if !ok {
		httpx.WriteError(w, r, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "serviço não configurado")
		return
	}
	rp.ServeHTTP(w, r)
}

func isCleanPath(p string) bool {
	c := path.Clean(p)
	return c == p || c+"/" == p
}

func bearerToken(r *http.Request) (token string, fromQuery bool) {
	const prefix = "Bearer "
	if h := r.Header.Get("Authorization"); len(h) > len(prefix) && strings.EqualFold(h[:len(prefix)], prefix) {
		return strings.TrimSpace(h[len(prefix):]), false
	}
	// Navegadores não enviam Authorization no handshake de WebSocket.
	if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return r.URL.Query().Get("access_token"), true
	}
	return "", false
}

// stripQueryToken evita que o token vá parar na URL/logs do serviço de destino.
func stripQueryToken(r *http.Request) {
	q := r.URL.Query()
	q.Del("access_token")
	r.URL.RawQuery = q.Encode()
}

func hasAnyRole(have, need []string) bool {
	if len(need) == 0 {
		return true
	}
	for _, n := range need {
		for _, h := range have {
			if h == n {
				return true
			}
		}
	}
	return false
}
