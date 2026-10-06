package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Rede-Medica-D-Excelencia-Vitalis/vitalis-service-template/internal/adapter/http"
	"github.com/Rede-Medica-D-Excelencia-Vitalis/vitalis-service-template/internal/adapter/postgres"
	redisx "github.com/Rede-Medica-D-Excelencia-Vitalis/vitalis-service-template/internal/adapter/redis"
	"github.com/Rede-Medica-D-Excelencia-Vitalis/vitalis-service-template/internal/app"
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
	log := logger.New(cfg.ServiceName, cfg.Env)

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

	repo := postgres.NewExampleRepo(pool)
	ob := outbox.NewWriter(pool)
	svc := app.NewExampleService(repo, ob, nil)

	router := httpserver.NewRouter(httpserver.Dependencies{
		ServiceName: cfg.ServiceName,
		Pool:        pool,
		Redis:       rdb,
		Examples:    svc,
	})

	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: router}
	go func() {
		log.Info("api listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("listen", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	log.Info("api stopped")
}
