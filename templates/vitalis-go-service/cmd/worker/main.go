package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Rede-Medica-D-Excelencia-Vitalis/vitalis-service-template/internal/adapter/postgres"
	redisx "github.com/Rede-Medica-D-Excelencia-Vitalis/vitalis-service-template/internal/adapter/redis"
	"github.com/Rede-Medica-D-Excelencia-Vitalis/vitalis-service-template/internal/config"
	"github.com/Rede-Medica-D-Excelencia-Vitalis/vitalis-service-template/internal/outbox"
	"github.com/Rede-Medica-D-Excelencia-Vitalis/vitalis-service-template/internal/platform/logger"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	log := logger.New(cfg.ServiceName+"-worker", cfg.Env)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("postgres", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	rdb, err := redisx.NewClient(cfg.RedisAddr, cfg.RedisPass, cfg.RedisDB)
	if err != nil {
		log.Error("redis", "err", err)
		os.Exit(1)
	}
	defer rdb.Close()

	stream := getenv("EVENTS_STREAM", "vitalis.events")
	pub := outbox.NewPublisher(pool, rdb, stream, log)
	log.Info("worker started", "stream", stream)
	pub.Run(ctx, 2*time.Second)
	log.Info("worker stopped")
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
