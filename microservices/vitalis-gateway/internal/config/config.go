package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config do Gateway: stateless, sem banco. Tudo vem de variável de ambiente.
type Config struct {
	ServiceName string
	HTTPAddr    string
	Env         string

	// JWKS do Identity: de onde o Gateway baixa as chaves públicas para validar o JWT.
	JWKSURL     string
	JWKSRefresh time.Duration
	JWTIssuer   string
	JWTAudience string

	CORSOrigins    []string
	RateLimitRPS   float64
	RateLimitBurst int
	MaxBodyBytes   int64

	UpstreamTimeout time.Duration
	ShutdownTimeout time.Duration

	// Upstreams: nome lógico do serviço -> URL base na rede interna.
	Upstreams map[string]string
}

func Load() Config {
	return Config{
		ServiceName: getenv("SERVICE_NAME", "vitalis-gateway"),
		HTTPAddr:    getenv("HTTP_ADDR", ":8080"),
		Env:         getenv("APP_ENV", "local"),

		JWKSURL:     getenv("JWKS_URL", "http://127.0.0.1:8081/v1/auth/jwks.json"),
		JWKSRefresh: time.Duration(getenvInt("JWKS_REFRESH_MIN", 10)) * time.Minute,
		JWTIssuer:   getenv("JWT_ISSUER", "vitalis-identity"),
		JWTAudience: getenv("JWT_AUDIENCE", "vitalis"),

		CORSOrigins:    splitCSV(getenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173")),
		RateLimitRPS:   float64(getenvInt("RATE_LIMIT_RPS", 20)),
		RateLimitBurst: getenvInt("RATE_LIMIT_BURST", 40),
		MaxBodyBytes:   int64(getenvInt("MAX_BODY_MB", 10)) << 20,

		UpstreamTimeout: time.Duration(getenvInt("UPSTREAM_TIMEOUT_SEC", 30)) * time.Second,
		ShutdownTimeout: time.Duration(getenvInt("SHUTDOWN_TIMEOUT_SEC", 10)) * time.Second,

		Upstreams: map[string]string{
			"identity": getenv("UPSTREAM_IDENTITY", "http://127.0.0.1:8081"),
			"clinical": getenv("UPSTREAM_CLINICAL", "http://127.0.0.1:8082"),
			"commerce": getenv("UPSTREAM_COMMERCE", "http://127.0.0.1:8083"),
			"orders":   getenv("UPSTREAM_ORDERS", "http://127.0.0.1:8084"),
			"delivery": getenv("UPSTREAM_DELIVERY", "http://127.0.0.1:8085"),
			"billing":  getenv("UPSTREAM_BILLING", "http://127.0.0.1:8086"),
			"comms":    getenv("UPSTREAM_COMMS", "http://127.0.0.1:8087"),
			"support":  getenv("UPSTREAM_SUPPORT", "http://127.0.0.1:8088"),
			"files":    getenv("UPSTREAM_FILES", "http://127.0.0.1:8089"),
			"search":   getenv("UPSTREAM_SEARCH", "http://127.0.0.1:8090"),
			"audit":    getenv("UPSTREAM_AUDIT", "http://127.0.0.1:8091"),
		},
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getenvInt(k string, def int) int {
	n, err := strconv.Atoi(os.Getenv(k))
	if err != nil {
		return def
	}
	return n
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
