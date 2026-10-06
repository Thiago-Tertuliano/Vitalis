package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config carrega variÃ¡veis de ambiente do serviÃ§o.
// Ao clonar o template, ajuste ServiceName / defaults no .env.example.
type Config struct {
	ServiceName string
	HTTPAddr    string
	Env         string

	DatabaseURL string
	RedisAddr   string
	RedisPass   string
	RedisDB     int

	ShutdownTimeout time.Duration

	JWTPrivateKeyPath string
	JWTPublicKeyPath  string
	JWTIssuer		  string
	JWTAudience       string
	AcessTTLMin       int
	RefreshTTLDays    int
}

func Load() (Config, error) {
	cfg := Config{
		ServiceName:     getenv("SERVICE_NAME", "vitalis-identity"),
		HTTPAddr:        getenv("HTTP_ADDR", ":8080"),
		Env:             getenv("APP_ENV", "local"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		RedisAddr:       getenv("REDIS_ADDR", "localhost:6379"),
		RedisPass:       os.Getenv("REDIS_PASSWORD"),
		RedisDB:         getenvInt("REDIS_DB", 0),
		ShutdownTimeout: time.Duration(getenvInt("SHUTDOWN_TIMEOUT_SEC", 10)) * time.Second,
		JWTPrivateKeyPath: getenv("JWT_PRIVATE_KEY_PATH", "./secrets/jwt_private.pem"),
		JWTPublicKeyPath: getenv("JWT_PUBLIC_KEY_PATH", "./secrets/jwt_public.pem"),
		JWTIssuer: getenv("JWT_ISSUER", "vitalis-identity"),
		JWTAudience: getenv("JWT_AUDIENCE", "vitalis"),
		AcessTTLMin: getenvInt("JWT_ACCESS_TTL_MIN", 15),
		RefreshTTLDays: getenvInt("JWT_REFRESH_TTL_DAYS", 7),
	}
	if cfg.DatabaseURL == "" {
		return cfg, fmt.Errorf("DATABASE_URL Ã© obrigatÃ³rio")
	}
	return cfg, nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getenvInt(k string, def int) int {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

