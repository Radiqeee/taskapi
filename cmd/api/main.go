package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Radiqeee/taskapi/internal/auth"
	"github.com/Radiqeee/taskapi/internal/config"
	"github.com/Radiqeee/taskapi/internal/handler"
	"github.com/Radiqeee/taskapi/internal/store"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var st store.Store
	if cfg.DatabaseURL != "" {
		pg, err := store.NewPostgres(ctx, cfg.DatabaseURL)
		if err != nil {
			return err
		}
		defer pg.Close()
		st = pg
		slog.Info("using postgres storage")
	} else {
		st = store.NewMemory()
		slog.Warn("DATABASE_URL is not set, using in-memory storage (data is lost on restart)")
	}
	if cfg.JWTSecret == config.DefaultJWTSecret {
		slog.Warn("JWT_SECRET is not set, using an insecure default")
	}

	h := handler.New(st, auth.NewTokenManager(cfg.JWTSecret, 24*time.Hour))
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           h.Router(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
