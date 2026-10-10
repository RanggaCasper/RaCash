package event

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/clock"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/queue"
)

const maxRelayAttempts = 10

// Relay memindahkan event PENDING dari outbox ke Asynq.
type Relay struct {
	db       *gorm.DB
	client   *asynq.Client
	registry *Registry
	clock    clock.Clock
	log      *slog.Logger
	batch    int
	interval time.Duration
}

func NewRelay(db *gorm.DB, client *asynq.Client, reg *Registry, c clock.Clock, log *slog.Logger, batch int, interval time.Duration) *Relay {
	return &Relay{db: db, client: client, registry: reg, clock: c, log: log.With("component", "outbox-relay"), batch: batch, interval: interval}
}

// Run berjalan sampai ctx dibatalkan.
func (r *Relay) Run(ctx context.Context) {
	t := time.NewTicker(r.interval)
	defer t.Stop()
	for {
		for {
			n, err := r.ProcessBatch(ctx)
			if err != nil && !errors.Is(err, context.Canceled) {
				r.log.Error("relay batch gagal", "err", err)
			}
			if err != nil || n < r.batch {
				break // batch tidak penuh: tunggu tick berikutnya
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// ProcessBatch memproses satu batch; mengembalikan jumlah event yang diambil.
// SKIP LOCKED membuat beberapa replika worker aman berjalan bersamaan.
func (r *Relay) ProcessBatch(ctx context.Context) (int, error) {
	var n int
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := r.clock.Now()
		var recs []outboxRecord
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status = ? AND available_at <= ?", StatusPending, now).
			Order("available_at").Limit(r.batch).Find(&recs).Error; err != nil {
			return err
		}
		n = len(recs)
		for i := range recs {
			rec := &recs[i]
			if err := r.dispatch(ctx, rec); err != nil {
				r.markRetry(rec, now, err)
			} else {
				rec.Status, rec.PublishedAt, rec.LastError = StatusPublished, &now, nil
			}
			if err := tx.Model(rec).Select("status", "attempts", "last_error", "available_at", "published_at").
				Updates(rec).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return n, err
}

func (r *Relay) dispatch(ctx context.Context, rec *outboxRecord) error {
	for _, sub := range r.registry.Subscribers(rec.EventType) {
		task := asynq.NewTask(TaskType(rec.EventType, sub), rec.Payload)
		_, err := r.client.EnqueueContext(ctx, task,
			asynq.TaskID(rec.ID.String()+":"+sub),
			asynq.Queue(queue.QueueDefault),
			asynq.MaxRetry(10),
			asynq.Retention(24*time.Hour),
		)
		if err != nil && !errors.Is(err, asynq.ErrTaskIDConflict) && !errors.Is(err, asynq.ErrDuplicateTask) {
			return err
		}
	}
	return nil
}

func (r *Relay) markRetry(rec *outboxRecord, now time.Time, err error) {
	msg := err.Error()
	rec.Attempts++
	rec.LastError = &msg
	if rec.Attempts >= maxRelayAttempts {
		rec.Status = StatusFailed
		r.log.Error("event outbox gagal permanen", "event_id", rec.ID, "type", rec.EventType, "err", err)
		return
	}
	backoff := time.Duration(1<<rec.Attempts) * time.Second
	rec.AvailableAt = now.Add(backoff)
}
