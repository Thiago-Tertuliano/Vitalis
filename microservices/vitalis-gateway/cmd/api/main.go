package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-gateway/internal/auth"
	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-gateway/internal/config"
	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-gateway/internal/gateway"
	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-gateway/internal/httpserver"
	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-gateway/internal/middleware"
	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-gateway/internal/platform/logger"
	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-gateway/internal/proxy"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()
	log := logger.New(cfg.ServiceName, cfg.Env)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// ---- autenticação ----
	jwks := auth.NewJWKS(cfg.JWKSURL, cfg.JWKSRefresh)
	initCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	if err := jwks.Refresh(initCtx); err != nil {
		// Identity pode subir depois do Gateway: só avisa, a busca é refeita sob demanda.
		log.Warn("JWKS indisponível no boot", "url", cfg.JWKSURL, "err", err)
	}
	cancel()
	validator := auth.NewValidator(jwks, cfg.JWTIssuer, cfg.JWTAudience)

	// ---- proxies para os serviços ----
	proxies, err := proxy.NewRegistry(cfg.Upstreams, cfg.UpstreamTimeout, log)
	if err != nil {
		log.Error("proxy", "err", err)
		os.Exit(1)
	}

	router := httpserver.NewRouter(httpserver.Dependencies{
		ServiceName:  cfg.ServiceName,
		Log:          log,
		JWKS:         jwks,
		Gateway:      gateway.NewHandler(validator, proxies, log),
		CORSOrigins:  cfg.CORSOrigins,
		RateLimiter:  middleware.NewRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst),
		MaxBodyBytes: cfg.MaxBodyBytes,
	})

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		// Sem WriteTimeout: ele derrubaria conexões WebSocket longas (/v1/realtime).
	}
	go func() {
		log.Info("gateway listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("listen", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancelShutdown()
	_ = srv.Shutdown(shutdownCtx)
	log.Info("gateway stopped")
}
