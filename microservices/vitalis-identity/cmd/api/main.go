package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-identity/internal/adapter/crypto"
	httpserver "github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-identity/internal/adapter/http"
	jwtadapter "github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-identity/internal/adapter/jwt"
	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-identity/internal/adapter/postgres"
	redisx "github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-identity/internal/adapter/redis"
	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-identity/internal/app"
	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-identity/internal/config"
	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-identity/internal/outbox"
	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-identity/internal/platform/logger"
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

	// ---- adapters ----
	users := postgres.NewUserRepo(pool)
	refresh := postgres.NewRefreshRepo(pool)
	addrs := postgres.NewAddressRepo(pool)
	resets := postgres.NewResetRepo(pool)
	hasher := crypto.NewArgon2Hasher()
	ob := outbox.NewWriter(pool)

	tokens, err := jwtadapter.NewIssuer(
		cfg.JWTPrivateKeyPath,
		cfg.JWTPublicKeyPath,
		cfg.JWTIssuer,
		cfg.JWTAudience,
		cfg.AcessTTLMin, // campo com typo no config — ok se for int
		cfg.RefreshTTLDays,
		refresh,
	)
	if err != nil {
		log.Error("jwt issuer", "err", err)
		os.Exit(1)
	}

	// ---- use cases ----
	authSvc := app.NewAuthService(users, refresh, hasher, tokens, ob, nil)
	profileSvc := app.NewProfileService(users, ob)
	addressSvc := app.NewAddressService(addrs)
	adminSvc := app.NewAdminService(users, ob)
	passwordSvc := app.NewPasswordResetService(users, resets, hasher)

	router := httpserver.NewRouter(httpserver.Dependencies{
		ServiceName: cfg.ServiceName,
		Pool:        pool,
		Redis:       rdb,
		Auth:        authSvc,
		Profile:     profileSvc,
		Addresses:   addressSvc,
		Admin:       adminSvc,
		Passwords:   passwordSvc,
	})

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}
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
