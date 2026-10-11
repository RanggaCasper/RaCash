// Package queue membungkus Asynq: klien untuk enqueue, server worker, dan scheduler.
package queue

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
)

// Nama antrean dan bobot prioritasnya.
const (
	QueueCritical = "critical" // fulfillment, webhook lanjutan
	QueueDefault  = "default"  // event outbox, notifikasi
	QueueLow      = "low"      // sinkron harga, laporan, cleanup
)

// RedisOpt mengubah REDIS_URL menjadi opsi koneksi Asynq.
func RedisOpt(url string) (asynq.RedisConnOpt, error) {
	opt, err := asynq.ParseRedisURI(url)
	if err != nil {
		return nil, fmt.Errorf("queue: %w", err)
	}
	return opt, nil
}

func NewClient(opt asynq.RedisConnOpt) *asynq.Client { return asynq.NewClient(opt) }

// NewServer membuat server worker dengan prioritas antrean.
func NewServer(opt asynq.RedisConnOpt, concurrency int, log *slog.Logger) *asynq.Server {
	l := log.With("component", "asynq")
	return asynq.NewServer(opt, asynq.Config{
		Concurrency: concurrency,
		Queues:      map[string]int{QueueCritical: 6, QueueDefault: 3, QueueLow: 1},
		Logger:      asynqLogger{l},
		ErrorHandler: asynq.ErrorHandlerFunc(func(_ context.Context, t *asynq.Task, err error) {
			l.Error("task gagal", "type", t.Type(), "err", err)
		}),
	})
}

// NewScheduler membuat scheduler untuk job terjadwal (cron).
func NewScheduler(opt asynq.RedisConnOpt, log *slog.Logger) *asynq.Scheduler {
	return asynq.NewScheduler(opt, &asynq.SchedulerOpts{
		Logger:   asynqLogger{log.With("component", "asynq-scheduler")},
		Location: nil, // UTC; ekspresi cron ditulis dalam UTC
	})
}

type asynqLogger struct{ l *slog.Logger }

func (a asynqLogger) Debug(args ...any) { a.l.Debug(fmt.Sprint(args...)) }
func (a asynqLogger) Info(args ...any)  { a.l.Info(fmt.Sprint(args...)) }
func (a asynqLogger) Warn(args ...any)  { a.l.Warn(fmt.Sprint(args...)) }
func (a asynqLogger) Error(args ...any) { a.l.Error(fmt.Sprint(args...)) }
func (a asynqLogger) Fatal(args ...any) { a.l.Error(fmt.Sprint(args...)) }
