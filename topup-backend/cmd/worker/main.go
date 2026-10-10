// Command worker menjalankan Asynq worker, scheduler, dan outbox relay.
// Boleh dijalankan lebih dari satu replika.
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
		fmt.Fprintln(os.Stderr, "worker:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := logger.New(os.Stdout, cfg.LogLevel, cfg.AppEnv, "worker")
	slog.SetDefault(log)

	a, err := app.Bootstrap(cfg, log)
	if err != nil {
		return err
	}
	defer a.Close()

	w, err := a.Worker()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	metricsSrv := a.MetricsServer()
	go func() {
		if err := metricsSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("metrics server gagal", "err", err)
		}
	}()

	log.Info("worker berjalan", "concurrency", cfg.WorkerConcurrency, "metrics_addr", cfg.MetricsAddr)
	runErr := w.Run(ctx)
	log.Info("worker berhenti")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	return errors.Join(runErr, metricsSrv.Shutdown(shutdownCtx))
}
