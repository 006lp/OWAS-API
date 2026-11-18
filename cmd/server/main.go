package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"owas-api/internal/config"
	"owas-api/internal/infrastructure/upstream"
	"owas-api/internal/service"
	httptransport "owas-api/internal/transport/http"
)

const appVersion = "1.0.0"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load configuration", slog.Any("error", err))
		os.Exit(1)
	}

	// Wire infrastructure gateways -> application services -> Gin HTTP transport.
	storeGateway := upstream.NewStoreClient(cfg.Upstream.AppStoreBaseURL, cfg.Upstream.Timeout, nil)
	healthGateway := upstream.NewHealthClient(cfg.Upstream.HealthCloudBaseURL, cfg.Upstream.Timeout, nil)

	storeService := service.NewStoreService(storeGateway)
	healthService := service.NewHealthService(healthGateway)

	router := httptransport.NewRouter(httptransport.RouterDeps{
		StoreService:   storeService,
		HealthService:  healthService,
		DefaultProfile: cfg.DefaultProfile,
		Logger:         logger,
		Version:        appVersion,
		GinMode:        cfg.Server.GinMode,
	})

	srv := &http.Server{
		Addr:         cfg.Server.Addr(),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("starting OWAS-API server",
			slog.String("addr", cfg.Server.Addr()),
			slog.String("gin_mode", cfg.Server.GinMode),
			slog.String("version", appVersion),
		)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			logger.Error("server exited with error", slog.Any("error", err))
			os.Exit(1)
		}
	case <-ctx.Done():
		logger.Info("shutdown signal received, draining connections")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("graceful shutdown failed", slog.Any("error", err))
			os.Exit(1)
		}
		logger.Info("server stopped cleanly")
	}
}
