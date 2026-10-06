package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type ctxKey string

const (
	KeyRequestID ctxKey = "request_id"
	KeyUserID    ctxKey = "user_id"
	KeyRoles     ctxKey = "roles"
)

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			id = uuid.NewString()
		}
		w.Header().Set("X-Request-Id", id)
		ctx := context.WithValue(r.Context(), KeyRequestID, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// InternalIdentity lÃª headers injetados pelo Gateway (ADR-004).
// Em produÃ§Ã£o, sÃ³ aceite esses headers na rede interna.
func InternalIdentity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if uid := r.Header.Get("X-User-Id"); uid != "" {
			ctx = context.WithValue(ctx, KeyUserID, uid)
		}
		if roles := r.Header.Get("X-Roles"); roles != "" {
			ctx = context.WithValue(ctx, KeyRoles, roles)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequestIDFromCtx(ctx context.Context) string {
	if v, ok := ctx.Value(KeyRequestID).(string); ok {
		return v
	}
	return ""
}

func UserIDFromCtx(ctx context.Context) string {
	if v, ok := ctx.Value(KeyUserID).(string); ok {
		return v
	}
	return ""
}

func RolesFromCtx(ctx context.Context) string {
	if v, ok := ctx.Value(KeyRoles).(string); ok {
		return v
	}
	return ""
}

