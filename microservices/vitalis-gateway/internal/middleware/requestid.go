package middleware

import (
	"net/http"

	"github.com/google/uuid"
)

// RequestID garante um X-Request-Id em toda requisição, propagado aos serviços e devolvido ao cliente.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" || len(id) > 128 {
			id = uuid.NewString()
		}
		r.Header.Set("X-Request-Id", id)
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r)
	})
}
