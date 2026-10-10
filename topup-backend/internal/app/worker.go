package app

import (
	"context"
	"fmt"

	"github.com/hibiken/asynq"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/event"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/queue"
)

// Job platform (bukan milik modul mana pun).
const TaskIdempotencyCleanup = "platform:idempotency_cleanup"

// Worker menjalankan server Asynq, scheduler cron, dan outbox relay.
type Worker struct {
	server    *asynq.Server
	scheduler *asynq.Scheduler
	mux       *asynq.ServeMux
	relay     *event.Relay
}

// Worker merangkai handler job, subscriber event, dan jadwal dari semua modul.
func (a *App) Worker() (*Worker, error) {
	p := a.Platform
	mux := asynq.NewServeMux()
	bus := event.NewRegistry()
	sched := queue.NewScheduler(p.QueueOpt, p.Log)

	mux.HandleFunc(TaskIdempotencyCleanup, func(ctx context.Context, _ *asynq.Task) error {
		n, err := p.Idempotency.DeleteExpired(ctx)
		if err == nil && n > 0 {
			p.Log.Info("idempotency key kedaluwarsa dihapus", "count", n)
		}
		return err
	})
	if _, err := sched.Register("0 * * * *", asynq.NewTask(TaskIdempotencyCleanup, nil), asynq.Queue(queue.QueueLow)); err != nil {
		return nil, err
	}

	for _, m := range a.Registry.Modules() {
		m.RegisterJobs(mux)
		m.RegisterSubscribers(bus)
		if s, ok := m.(Scheduled); ok {
			if err := s.RegisterSchedules(sched); err != nil {
				return nil, fmt.Errorf("app: jadwal modul %s: %w", m.Name(), err)
			}
		}
	}
	bus.Mount(mux)

	return &Worker{
		server:    queue.NewServer(p.QueueOpt, p.Config.WorkerConcurrency, p.Log),
		scheduler: sched,
		mux:       mux,
		relay: event.NewRelay(p.DB, p.Queue, bus, p.Clock, p.Log,
			p.Config.OutboxBatchSize, p.Config.OutboxPollInterval),
	}, nil
}

// Run berjalan sampai ctx dibatalkan, lalu berhenti dengan rapi.
func (w *Worker) Run(ctx context.Context) error {
	if err := w.server.Start(w.mux); err != nil {
		return err
	}
	if err := w.scheduler.Start(); err != nil {
		w.server.Shutdown()
		return err
	}
	relayDone := make(chan struct{})
	go func() {
		w.relay.Run(ctx)
		close(relayDone)
	}()

	<-ctx.Done()
	<-relayDone
	w.scheduler.Shutdown()
	w.server.Shutdown()
	return nil
}
