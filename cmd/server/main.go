package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mymoments/internal/config"
	"mymoments/internal/httpapi"
	"mymoments/internal/logging"
	"mymoments/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	logFile, err := logging.NewRotatingWriter(cfg.LogDir, logging.MaxSizeBytes, logging.MaxBackups+1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "init file logging: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()
	slog.SetDefault(slog.New(slog.NewJSONHandler(io.MultiWriter(os.Stdout, logFile), &slog.HandlerOptions{Level: slog.LevelInfo})))
	slog.Info("startup", "step", "config-loaded", "addr", cfg.Addr, "data_dir", cfg.DataDir, "log_dir", cfg.LogDir, "db", cfg.DBPath())
	if err := run(cfg); err != nil {
		slog.Error("server exited", "err", err)
		os.Exit(1)
	}
}

func run(cfg config.Config) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	slog.Info("startup", "step", "db-open", "db", cfg.DBPath())
	st, err := store.Open(ctx, cfg.DSN())
	if err != nil {
		slog.Error("startup", "step", "db-open", "err", err)
		return err
	}
	defer st.Close()
	slog.Info("startup", "step", "db-ready")

	slog.Info("startup", "step", "http-init")
	api, err := httpapi.New(cfg, st)
	if err != nil {
		slog.Error("startup", "step", "http-init", "err", err)
		return err
	}
	slog.Info("startup", "step", "http-ready")
	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       90 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		slog.Info("startup", "step", "listening", "addr", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		if err != nil {
			return err
		}
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	slog.Info("server stopped")
	return nil
}
