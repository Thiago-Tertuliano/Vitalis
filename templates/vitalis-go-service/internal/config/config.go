package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config carrega variáveis de ambiente do serviço.
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
}

func Load() (Config, error) {
	cfg := Config{
		ServiceName:     getenv("SERVICE_NAME", "vitalis-service-template"),
		HTTPAddr:        getenv("HTTP_ADDR", ":8080"),
		Env:             getenv("APP_ENV", "local"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		RedisAddr:       getenv("REDIS_ADDR", "localhost:6379"),
		RedisPass:       os.Getenv("REDIS_PASSWORD"),
		RedisDB:         getenvInt("REDIS_DB", 0),
		ShutdownTimeout: time.Duration(getenvInt("SHUTDOWN_TIMEOUT_SEC", 10)) * time.Second,
	}
	if cfg.DatabaseURL == "" {
		return cfg, fmt.Errorf("DATABASE_URL é obrigatório")
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
