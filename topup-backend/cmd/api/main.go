// Command api menjalankan HTTP server publik/admin/H2H/webhook.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/app"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/config"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/logger"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "api:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := logger.New(os.Stdout, cfg.LogLevel, cfg.AppEnv, "api")
	slog.SetDefault(log)

	a, err := app.Bootstrap(cfg, log)
	if err != nil {
		return err
	}
	defer a.Close()

	srv := a.HTTP()
	metricsSrv := a.MetricsServer()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errc := make(chan error, 2)
	go func() { errc <- srv.Listen(cfg.HTTPAddr) }()
	go func() {
		if err := metricsSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errc <- err
		}
	}()
	log.Info("api berjalan", "addr", cfg.HTTPAddr, "metrics_addr", cfg.MetricsAddr)

	select {
	case <-ctx.Done():
	case err := <-errc:
		return err
	}

	log.Info("api berhenti")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	return errors.Join(srv.ShutdownWithContext(shutdownCtx), metricsSrv.Shutdown(shutdownCtx))
}
