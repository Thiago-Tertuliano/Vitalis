package httpx

import (
	"encoding/json"
	"net/http"
)

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteError usa o mesmo modelo Error do OpenAPI de todos os serviços Vitalis.
func WriteError(w http.ResponseWriter, r *http.Request, status int, code, msg string) {
	WriteJSON(w, status, map[string]any{
		"erro":       msg,
		"codigo":     code,
		"request_id": r.Header.Get("X-Request-Id"),
	})
}
